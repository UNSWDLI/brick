// Copyright 2020 Adam Chalkley
//
// https://github.com/atc0005/brick
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package files

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"text/template"

	"github.com/apex/log"

	"github.com/atc0005/go-ezproxy"
	"github.com/atc0005/go-ezproxy/activefile"

	"github.com/atc0005/brick/events"
	"github.com/atc0005/brick/internal/caller"
	"github.com/atc0005/brick/internal/fileutils"
)

func processRecord(record events.Record, notifyWorkQueue chan<- events.Record) {

	if record.Error != nil {
		log.Error(record.Error.Error())
	}

	// shouldn't encounter "loop variable XYZ captured by func literal" issue
	// because we're not in a loop (record isn't changing)
	go func() {
		notifyWorkQueue <- record
	}()

}

// ProcessDisableEvent receives a care-package of configuration settings, the
// original alert, a channel to send event records on and values representing
// the disabled users and reported user events log files. This function
// handles orchestration of multiple actions taken in response to the received
// alert and request to disable a user account (and disable the associated
// sessions). This function returns a collection of
//
// TODO: This function and those called within are *badly* in need of
// refactoring.
func ProcessDisableEvent(
	alert events.SplunkAlertEvent,
	disabledUsers *DisabledUsers,
	reportedUserEventsLog *ReportedUserEventsLog,
	ignoredSources IgnoredSources,
	notifyWorkQueue chan<- events.Record,
	terminateSessions bool,
	ezproxyActiveFilePath string,
	ezproxySessionsSearchDelay int,
	ezproxySessionSearchRetries int,
	ezproxyExecutable string,
) {

	// Record/log that a username was reported
	//
	// It so happens that we are going to try and disable a username. The
	// assumption with this remark is that the remote monitoring system is
	// just passing along specific information to a specific endpoint, but in
	// reality we're setting up an alert in the monitoring system with a
	// specific outcome in mind.
	disableRequestReceivedResult := logEventDisableRequestReceived(
		alert,
		reportedUserEventsLog,
	)

	processRecord(disableRequestReceivedResult, notifyWorkQueue)

	// check whether username or IP Address is ignored, return early if true
	// or if there is an error looking up the status which the sysadmin did
	// not opt to disregard.
	ignoredEntryFound, ignoredEntryResults := isIgnored(alert, reportedUserEventsLog, ignoredSources)
	switch {
	case ignoredEntryResults.Error != nil:

		if ignoredSources.IgnoreLookupErrors {
			// If sysadmin opted to ignore lookup errors then honor the
			// request; emit complaint (to console, local logs, syslog via
			// systemd, etc) and ignore the lookup error by proceeding.
			//
			// WARNING: See GH-62; this "feature" may be removed in a future
			// release in order to avoid potentially unexpected logic bugs.
			log.Warn(ignoredEntryResults.Error.Error())
			break
		}

		// send record for notification
		processRecord(ignoredEntryResults, notifyWorkQueue)

		// exit after sending notification
		return

	// early exit to force desired ignore behavior
	case ignoredEntryFound:

		// Note: `logEventIgnoredUsername()` is called within `isIgnored()`,
		// so we refrain from calling it again explicitly here.
		processRecord(ignoredEntryResults, notifyWorkQueue)

		// exit after sending notification
		return

	}

	// check to see if username has already been disabled
	disabledUserEntry := alert.Username + disabledUsers.EntrySuffix
	disableEntryFound, disableEntryLookupErr := fileutils.HasLine(
		disabledUserEntry,
		"#",
		disabledUsers.FilePath,
	)

	// Handle logic for disabling user account
	switch {

	case disableEntryLookupErr != nil:

		errMsg := fmt.Errorf(
			"error while checking disabled status for user %q from IP %q: %w",
			alert.Username,
			alert.UserIP,
			disableEntryLookupErr,
		)

		if ignoredSources.IgnoreLookupErrors {
			// If sysadmin opted to ignore lookup errors then honor the
			// request; emit complaint (to console, local logs, syslog via
			// systemd, etc) and ignore the lookup error by proceeding.
			//
			// WARNING: See GH-62; this "feature" may be removed in a future
			// release in order to avoid potentially unexpected logic bugs.
			log.Warn(disableEntryLookupErr.Error())

			// NOTE: If the lookup error is being ignored, we skip all
			// attempts to disable the user account.
			break
		}

		result := events.NewRecord(
			alert,
			errMsg,
			// FIXME: Not sure what Note or "summary" field value to use here
			"",
			events.ActionFailureDisabledUsername,
			nil,
		)

		processRecord(result, notifyWorkQueue)

		return

	case !disableEntryFound:

		// log our intent to disable the username
		logEventDisablingUsername(alert, reportedUserEventsLog)

		// disable usename
		if err := disableUser(alert, disabledUsers); err != nil {
			result := events.NewRecord(
				alert,
				err,
				// FIXME: Unsure what note to use here
				"",
				events.ActionFailureDisabledUsername,
				nil,
			)

			processRecord(result, notifyWorkQueue)

			return
		}

		// log success (file, notifications, etc.)
		disableUsernameResult := logEventDisabledUsername(alert, reportedUserEventsLog)
		processRecord(disableUsernameResult, notifyWorkQueue)

	case disableEntryFound:

		usernameAlreadyDisabledResult := logEventUsernameAlreadyDisabled(alert, reportedUserEventsLog)
		processRecord(usernameAlreadyDisabledResult, notifyWorkQueue)

	}

	// At this point the username has been disabled, either just now or as
	// part of a previous report. We should proceed with session termination
	// if enabled or note that the setting is not enabled for troubleshooting
	// purposes later.
	switch {
	case !terminateSessions:

		log.Warn("Sessions termination is disabled via configuration setting. Sessions will persist until they timeout.")

		userSessions, userSessionsLookupErr := getUserSessions(
			alert,
			reportedUserEventsLog,
			ezproxyActiveFilePath,
			ezproxySessionsSearchDelay,
			ezproxySessionSearchRetries,
			ezproxyExecutable,
		)

		if userSessionsLookupErr != nil {
			record := events.NewRecord(
				alert,
				userSessionsLookupErr,
				"",
				events.ActionFailureUserSessionLookupFailure,
				nil,
			)

			processRecord(record, notifyWorkQueue)

		}

		var userSessionIDs []string
		for _, session := range userSessions {
			userSessionIDs = append(userSessionIDs, session.SessionID)
		}

		sessionsSkipped := strings.Join(userSessionIDs, `", "`)

		sessionsSkippedMsg := fmt.Sprintf(
			`Skipping termination of sessions: "%s"`,
			sessionsSkipped,
		)

		log.Warn(sessionsSkippedMsg)

		record := events.NewRecord(
			alert,
			nil,
			sessionsSkippedMsg,
			events.ActionSkippedTerminateUserSessions,
			nil,
		)

		processRecord(record, notifyWorkQueue)

	case terminateSessions:

		userSessions, userSessionsLookupErr := getUserSessions(
			alert,
			reportedUserEventsLog,
			ezproxyActiveFilePath,
			ezproxySessionsSearchDelay,
			ezproxySessionSearchRetries,
			ezproxyExecutable,
		)

		if userSessionsLookupErr != nil {
			record := events.NewRecord(
				alert,
				userSessionsLookupErr,
				"",
				events.ActionFailureUserSessionLookupFailure,
				nil,
			)

			processRecord(record, notifyWorkQueue)

		}

		// logEventTerminatingUserSession is called within this function for
		// each session termination attempt (one or many) and
		// logEventTerminatedUserSessions is called at the end of the function
		// to provide a summary of the results.
		terminateUserSessionsResult := terminateUserSessions(
			alert,
			reportedUserEventsLog,
			userSessions,
			ezproxyActiveFilePath,
			ezproxyExecutable,
		)

		processRecord(terminateUserSessionsResult, notifyWorkQueue)

	}

}

// isIgnored is a wrapper function to help concentrate common ignored status
// checks in one place. If there are issues checking ignored status,
// explicitly state that the username or IP Address is ignored and return the
// error. The caller can then apply other logic to determine how the error
// condition should be treated.
func isIgnored(
	alert events.SplunkAlertEvent,
	reportedUserEventsLog *ReportedUserEventsLog,
	ignoredSources IgnoredSources,
) (bool, events.Record) {

	ignoredUserEntryFound, ignoredUserLookupErr := fileutils.HasLine(
		alert.Username,
		"#",
		ignoredSources.IgnoredUsersFile,
	)

	if ignoredUserLookupErr != nil {

		errMsg := fmt.Errorf(
			"error while checking ignored status for user %q from IP %q: %w",
			alert.Username,
			alert.UserIP,
			ignoredUserLookupErr,
		)

		result := events.NewRecord(
			alert,
			errMsg,
			// FIXME: Unsure what note to add here
			"",
			events.ActionFailureIgnoredUsername,
			nil,
		)

		// on error, assume username or IP should be ignored
		return true, result

	}

	if ignoredUserEntryFound {
		ignoredUsernameResult := logEventIgnoredUsername(
			alert,
			reportedUserEventsLog,
			ignoredSources.IgnoredUsersFile,
		)

		return true, ignoredUsernameResult

	}

	// check to see if IP Address has been ignored
	ipAddressIgnoreEntryFound, ipAddressIgnoreLookupErr := fileutils.HasLine(
		alert.UserIP,
		"#",
		ignoredSources.IgnoredIPAddressesFile,
	)

	if ipAddressIgnoreLookupErr != nil {

		errMsg := fmt.Errorf(
			"error while checking ignored status for IP %q associated with user %q: %w",
			alert.UserIP,
			alert.Username,
			ipAddressIgnoreLookupErr,
		)

		result := events.NewRecord(
			alert,
			errMsg,
			// FIXME: Unsure what note to add here
			"",
			events.ActionFailureIgnoredIPAddress,
			nil,
		)

		// on error, assume username or IP should be ignored
		return true, result
	}

	if ipAddressIgnoreEntryFound {

		ignoredIPAddressResult := logEventIgnoredIPAddress(
			alert,
			reportedUserEventsLog,
			ignoredSources.IgnoredIPAddressesFile,
		)

		return true, ignoredIPAddressResult

	}

	// the username and associated IP Addr is *not* ignored if:
	//
	// - no error occurs looking up the ignored status
	// - no match is found

	// FIXME: Not a fan of returning an empty Record here. If we drop Records
	// directly into the notifyWorkQueue channel instead of passing up this is
	// no longer necessary.
	return false, events.Record{}

}

func getUserSessions(
	alert events.SplunkAlertEvent,
	reportedUserEventsLog *ReportedUserEventsLog,
	ezproxyActiveFilePath string,
	ezproxySessionsSearchDelay int,
	ezproxySessionSearchRetries int,
	ezproxyExecutable string,
) (ezproxy.UserSessions, error) {

	reader, readerErr := activefile.NewReader(alert.Username, ezproxyActiveFilePath)
	if readerErr != nil {
		activeFileReaderErr := fmt.Errorf(
			"error while creating activeFile reader to retrieve sessions associated with user %q: %w",
			alert.Username,
			readerErr,
		)

		return nil, activeFileReaderErr
	}

	// Adjust stubbornness of newly created reader (overridding
	// library/package default values with our own)
	if err := reader.SetSearchDelay(ezproxySessionsSearchDelay); err != nil {
		searchDelayErr := fmt.Errorf(
			"error while setting search delay for activeFile reader to retrieve sessions associated with user %q: %w",
			alert.Username,
			err,
		)

		return nil, searchDelayErr

	}

	if err := reader.SetSearchRetries(ezproxySessionSearchRetries); err != nil {
		searchRetriesErr := fmt.Errorf(
			"error while setting search retries for activeFile reader to retrieve sessions associated with user %q: %w",
			alert.Username,
			err,
		)

		return nil, searchRetriesErr
	}

	log.Debugf(
		"%s: Searching %q for %q",
		caller.GetFuncName(),
		ezproxyActiveFilePath,
		alert.Username,
	)

	activeSessions, userSessionsLookupErr := reader.MatchingUserSessions()
	if userSessionsLookupErr != nil {
		userSessionsRetrievalErr := fmt.Errorf(
			"error retrieving matching user sessions associated with user %q: %w",
			alert.Username,
			userSessionsLookupErr,
		)

		return nil, userSessionsRetrievalErr
	}

	return activeSessions, nil
}

func terminateUserSessions(
	alert events.SplunkAlertEvent,
	reportedUserEventsLog *ReportedUserEventsLog,
	activeSessions ezproxy.UserSessions,
	ezproxyActiveFilePath string,
	ezproxyExecutable string,
) events.Record {

	// TODO: On the fence re emitting this output each time
	// log.Debug("ProcessEvent: Session termination enabled")
	log.Info("Session termination enabled")

	// build sessions list specific to provided user and active file using
	// an ezproxy.SessionsReader

	// If we received an alert from monitoring systems, there *should* be
	// at least one user session active in order for the alert to have
	// been generated in the first place. If not, we are considering that
	// an error.
	//
	// NOTE: The atc0005/go-ezproxy package performs retries per our above
	// configuration, so this session count "error" is *after* we have
	// already retried a set number of times; retries are performed in
	// case there is a race condition between EZproxy creating the session
	// and our receiving the notification.
	if len(activeSessions) == 0 {

		activeSessionsCountErr := fmt.Errorf(
			"0 active sessions found for username %q in file %q",
			alert.Username,
			ezproxyActiveFilePath,
		)

		return events.NewRecord(
			alert,
			activeSessionsCountErr,
			"",
			events.ActionFailureTerminatedUserSession,
			nil,
		)
	}

	log.Infof(
		"%d active sessions found for %q",
		len(activeSessions),
		alert.Username,
	)

	for _, session := range activeSessions {
		logEventTerminatingUserSession(alert, session)
	}

	terminationResults := activeSessions.Terminate(ezproxyExecutable)

	// User sessions *should* now be terminated; results of the attempts
	// are recorded for further review to confirm.

	logTerminatedUserSessionsResult := logEventTerminatedUserSessions(
		alert,
		reportedUserEventsLog,
		terminationResults,
	)

	return logTerminatedUserSessionsResult

}

// disableUser adds the specified username to the disabled users file. This
// function is intended to be called from within another function that first
// confirms that the specified user account has not already been disabled.
func disableUser(alert events.SplunkAlertEvent, disabledUsers *DisabledUsers) error {

	// NOTE: Notifications are handled by the caller

	log.Debug("DisableUser: disabling user per alert")
	if err := appendToFile(
		fileEntry{
			Alert:       alert,
			EntrySuffix: disabledUsers.EntrySuffix,
		},
		disabledUsers.Template,
		disabledUsers.FilePath,
		disabledUsers.FilePermissions,
	); err != nil {
		return fmt.Errorf(
			"error updating disabled user file %q: %w",
			disabledUsers.FilePath,
			err,
		)
	}

	return nil

}

// appendToFile is a helper function that accepts a new message, a destination
// filename and intended permissions for the filename if it does not already
// exist. All leading and trailing whitespace is removed from the new message
// and one trailing newline appended.
func appendToFile(entry fileEntry, tmpl *template.Template, filename string, perms os.FileMode) error {

	var mutex = &sync.Mutex{}

	log.Debugf("Attempting to open %q", filename)

	// If the file doesn't exist, create it, or append to the file
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, perms)
	if err != nil {
		return fmt.Errorf("error encountered opening file %q: %w", filename, err)
	}
	defer f.Close()
	log.Debugf("Successfully opened %q", filename)

	log.Debug("Locking mutex")
	mutex.Lock()
	defer func() {
		log.Debug("Unlocking mutex from deferred anonymous func")
		mutex.Unlock()
	}()

	log.Debugf("Executing template to update %q", filename)
	if err := tmpl.Execute(f, entry); err != nil {
		f.Close() // ignore error; Write error takes precedence
		return fmt.Errorf("error writing to file %q: %w", filename, err)
	}
	log.Debugf("Successfully executed template to update %q", filename)

	log.Debug("Syncing file modifications")
	if err := f.Sync(); err != nil {
		return fmt.Errorf(
			"failed to explicitly sync file %q after writing: %s",
			filename,
			err,
		)
	}
	log.Debugf("Successfully synced modifications to %q", filename)

	log.Debugf("Closing %q", filename)
	if err := f.Close(); err != nil {
		return fmt.Errorf("error closing file %q: %w", filename, err)
	}
	log.Debugf("Successfully closed %q", filename)

	return nil
}

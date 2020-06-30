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

package config

import (
	"os"

	"github.com/alexflint/go-arg"
)

// Config is a unified set of configuration values for this application. This
// struct is configured via command-line flags or TOML configuration file
// provided by the user. The majority of values held by this object are
// intended to be retrieved via "Getter" methods.
type Config struct {

	// Use our template, define distinct collections of configuration settings
	// TODO: I do not know whether struct tags are needed (or even allowed)
	// here for embedded structs.
	cliConfig  configTemplate
	fileConfig configTemplate

	flagParser *arg.Parser `toml:"-" arg:"-"`
}

// Network is a collection of network-related settings provided via CLI and
// config file sources.
type Network struct {

	// LocalTCPPort is the TCP port that this application should listen on for
	// incoming requests
	LocalTCPPort *int `toml:"local_tcp_port" arg:"--port,env:BRICK_LOCAL_TCP_PORT" help:"TCP port that this application should listen on for incoming HTTP requests."`

	// LocalIPAddress is the IP Address that this application should listen on
	// for incoming requests
	LocalIPAddress *string `toml:"local_ip_address" arg:"--ip-address,env:BRICK_LOCAL_IP_ADDRESS" help:"Local IP Address that this application should listen on for incoming HTTP requests."`
}

// Logging is a collection of logging-related settings provided via CLI and
// config file sources.
type Logging struct {
	// Level is the chosen logging level
	Level *string `toml:"level" arg:"--log-level,env:BRICK_LOG_LEVEL" help:"Log message priority filter. Log messages with a lower level are ignored."`

	// Output is one of the standard application outputs, stdout or stderr
	Output *string `toml:"output" arg:"--log-output,env:BRICK_LOG_OUTPUT" help:"Log messages are written to this output target."`

	// LogFormat controls which output format is used for log messages
	// generated by this application. This value is from a smaller subset
	// of the formats supported by the third-party leveled-logging package
	// used by this application.
	Format *string `toml:"format" arg:"--log-format,env:BRICK_LOG_FORMAT" help:"Log messages are written in this format."`
}

// DisabledUsers represents the path to, and permissions for, the file
// generated by this application containing disabled accounts that EZproxy
// should import and disabled from logging in.
type DisabledUsers struct {

	// File is the fully-qualified path to the EZproxy include file where this
	// application should write disabled user accounts.
	File *string `toml:"file_path" arg:"--disabled-users-file,env:BRICK_DISABLED_USERS_FILE" help:"fully-qualified path to the EZproxy include file where this application should write disabled user accounts."`

	// EntrySuffix is the string that is appended after every username added
	// to the disabled users file in order to deny login access.
	EntrySuffix *string `toml:"entry_suffix" arg:"--disabled-users-entry-suffix,env:BRICK_DISABLED_USERS_ENTRY_SUFFIX" help:"The string that is appended after every username added to the disabled users file in order to deny login access."`

	// Permissions is the desired file permissions when this file is created.
	// Note: The ezproxy daemon will need to be able to read this file.
	FilePermissions *os.FileMode `toml:"file_permissions" arg:"--disabled-users-file-perms,env:BRICK_DISABLED_USERS_FILE_PERMISSIONS" help:"Desired file permissions when this file is created. Note: The ezproxy daemon will need to be able to read this file."`
}

// ReportedUsers represents the path to, and permissions for, the file
// generated by this application for review by fail2ban for potential IP-ban
// actions and humans alike.
type ReportedUsers struct {

	// File is the fully-qualified path to the log file where this application
	// should log user disable request events for fail2ban to ingest.
	LogFile *string `toml:"file_path" arg:"--reported-users-log-file,env:BRICK_REPORTED_USERS_LOG_FILE" help:"Fully-qualified path to the log file where this application should log user disable request events for fail2ban to ingest."`

	// Permissions is the desired file permissions when this file is created.
	// Note: fail2ban will need to be able to read this file.
	LogFilePermissions *os.FileMode `toml:"file_permissions" arg:"--reported-users-log-file-perms,env:BRICK_REPORTED_USERS_LOG_FILE_PERMISSIONS" help:"Desired file permissions when this file is created. Note: fail2ban will need to be able to read this file."`
}

// IgnoredUsers represents the fully-qualified path to the file containing a
// list of user accounts which should not be disabled and whose associated IP
// should not be banned by this application. Note: The same IP could end up
// being (temporarily) banned by association with another user account which
// is not in the list of user accounts to be ignored. See also the option to
// ignore specific (individual) IPs.
type IgnoredUsers struct {

	// File is the fully-qualified path to the file containing a list of user
	// accounts that should not be disabled.
	File *string `toml:"file_path" arg:"--ignored-users-file,env:BRICK_IGNORED_USERS_FILE" help:"Fully-qualified path to the file containing a list of user accounts which should not be disabled and whose IP Address reported in the same alert should not be disabled by this application. Leading and trailing whitespace per line is ignored."`
}

// IgnoredIPAddresses represents the fully-qualified path to the file
// containing a list of individual IP Addresses which should not be banned by
// this application. User accounts associated with the individual IP Addresses
// in this file are not disabled by this application when reported in the same
// alert, though report user accounts can still be disabled when associated
// with a different IP Address. See also the option to ignore specific
// user accounts.
type IgnoredIPAddresses struct {

	// File is the fully-qualified path to the file containing a list of
	// individual IP Addresses which should not be banned by this application.
	File *string `toml:"file_path" arg:"--ignored-ips-file,env:BRICK_IGNORED_IP_ADDRESSES_FILE" help:"Fully-qualified path to the file containing a list of individual IP Addresses which should not be disabled and which user account reported in the same alert should not be disabled by this application. Leading and trailing whitespace per line is ignored."`
}

// MSTeams represents the various configuration settings used to send
// notifications to a Microsoft Teams channel.
type MSTeams struct {

	// WebhookURL is the full URL used to submit messages to the Teams
	// channel. This URL is in the form of
	// https://outlook.office.com/webhook/xxx or
	// https://outlook.office365.com/webhook/xxx. This URL needs to be created
	// in advance by adding/configuring a Webhook Connector in a Microsoft
	// Teams channel that you wish to submit messages to using this
	// application.
	WebhookURL *string `toml:"webhook_url" arg:"--teams-webhook-url,env:BRICK_MSTEAMS_WEBHOOK_URL" help:"The Webhook URL provided by a preconfigured Connector. If specified, this application will attempt to send client request details to the Microsoft Teams channel associated with the webhook URL."`

	// Delay is the number of seconds to wait between Microsoft Teams message
	// delivery attempts.
	Delay *int `toml:"delay" arg:"--teams-notify-delay,env:BRICK_MSTEAMS_WEBHOOK_DELAY" help:"The number of seconds to wait between Microsoft Teams message delivery attempts."`

	// Retries is the number of attempts that this application will make to
	// deliver Microsoft Teams messages before giving up.
	Retries *int `toml:"retries" arg:"--teams-notify-retries,env:BRICK_MSTEAMS_WEBHOOK_RETRIES" help:"The number of attempts that this application will make to deliver Microsoft Teams messages before giving up."`
}

// EZproxy represents that various configuration settings used to interact
// with EZproxy and files/settings used by EZproxy.
type EZproxy struct {

	// ExecutablePath is the fully-qualified path to the EZproxy
	// executable/binary. This executable is usually named 'ezproxy' and is
	// set to start at system boot. The fully-qualified path to this
	// executable is required for session termination.
	ExecutablePath *string `toml:"executable_path" arg:"--ezproxy-executable-path,env:BRICK_EZPROXY_EXECUTABLE_PATH" help:"The fully-qualified path to the EZproxy executable/binary. This executable is usually named 'ezproxy' and is set to start at system boot. The fully-qualified path to this executable is required for session termination."`

	// ActiveFilePath is the fully-qualified path to the Active Users and
	// Hosts "state" file used by EZproxy (and this application) to track
	// current sessions and hosts managed by EZproxy.
	ActiveFilePath *string `toml:"active_file_path" arg:"--ezproxy-active-file-path,env:BRICK_EZPROXY_ACTIVE_FILE_PATH" help:"The fully-qualified path to the Active Users and Hosts 'state' file used by EZproxy (and this application) to track current sessions and hosts managed by EZproxy."`

	// AuditFileDirPath is the path to the directory containing the EZproxy
	// audit files. The assumption is made that all files within are based on
	// YYYYMMDD.txt pattern. Any other file pattern found within this path is
	// ignored (e.g, .zip or .tar or whatnot for a one-off quick backup made
	// by a sysadmin of a specific file).
	AuditFileDirPath *string `toml:"audit_file_dir_path" arg:"--ezproxy-audit-file-dir-path,env:BRICK_EZPROXY_AUDIT_FILE_DIR_PATH" help:"The path to the directory containing the EZproxy audit files. The assumption is made that all files within are based on YYYYMMDD.txt pattern. Any other file pattern found within this path is ignored (e.g, .zip or .tar or whatnot for a one-off quick backup made by a sysadmin of a specific file)."`

	// SearchRetries is the number of retries allowed for the audit log and
	// active files before the application accepts that "cannot find matching
	// session IDs for specific user" is really the truth of it and not a race
	// condition between this application and the EZproxy application (e.g.,
	// EZproxy accepts a login, but delays writing the state information for
	// about 2 seconds to keep from hammering the storage device).
	SearchRetries *int `toml:"search_retries" arg:"--ezproxy-search-retries,env:BRICK_EZPROXY_SEARCH_RETRIES" help:"The number of retries allowed for the audit log and active files before the application accepts that 'cannot find matching session IDs for specific user' is really the truth of it and not a race condition between this application and the EZproxy application (e.g., EZproxy accepts a login, but delays writing the state information for about 2 seconds to keep from hammering the storage device)."`

	// SearchDelay is the delay in seconds between searches of the audit log
	// or active file for a specified username. This is an attempt to work
	// around race conditions between EZproxy updating its state file (which
	// has been observed to have a delay of up to several seconds) and this
	// application *reading* the active file. This delay is applied to the
	// initial search and each subsequent retried search for the provided
	// username.
	SearchDelay *int `toml:"search_delay" arg:"--ezproxy-search-delay,env:BRICK_EZPROXY_SEARCH_DELAY" help:"The delay in seconds between searches of the audit log or active file for a specified username. This is an attempt to work around race conditions between EZproxy updating its state file (which has been observed to have a delay of up to several seconds) and this application *reading* the active file. This delay is applied to the initial search and each subsequent retried search for the provided username."`

	// TerminateSessions controls whether session termination support is
	// enabled.
	//
	// If false, session termination will not be initiated by this
	// application, though current session IDs found as part of preparing for
	// termination will still be logged for troubleshooting purposes.
	//
	// If setting (or leaving) this as false, the assumption is that either no
	// handling of reported users is desired (other than perhaps logging and
	// notification) or that a tool such as fail2ban is used to monitor the
	// reported users log file and temporarily block the source IP in order to
	// force session timeout.
	TerminateSessions *bool `toml:"terminate_sessions" arg:"--ezproxy-terminate-sessions,env:BRICK_EZPROXY_TERMINATE_SESSIONS" help:"Whether session termination support is enabled. If false, session termination will not be initiated by this application, though current session IDs found as part of preparing for termination will still be logged for troubleshooting purposes. 	// If setting (or leaving) this as false, the assumption is that either no handling of reported users is desired (other than perhaps logging and notification) or that a tool such as fail2ban is used to monitor the reported users log file and temporarily block the source IP in order to force session timeout."`
}

// configTemplate is our base configuration template used to collect values
// specified by various configuration sources. This template struct is
// embedded within the main Config struct once for each config source.
type configTemplate struct {

	// Embed other structs referenced directly by TOML config file here.
	// Provide an explicit name in an effort to setup "namespacing" as a
	// logical arrangement.
	Network
	Logging
	DisabledUsers
	ReportedUsers
	IgnoredUsers
	IgnoredIPAddresses
	MSTeams
	EZproxy

	IgnoreLookupErrors *bool `toml:"ignore_lookup_errors" arg:"--ignore-lookup-errors,env:BRICK_IGNORE_LOOKUP_ERRORS" help:"Whether application should continue if attempts to lookup existing disabled or ignored status for a username or IP Address fail."`

	// ConfigFile represents the fully-qualified path to a configuration file
	// consulted for settings not provided via CLI flags
	ConfigFile *string `toml:"-" arg:"--config-file,env:BRICK_CONFIG_FILE" help:"Full path to optional TOML-formatted configuration file. See contrib/brick/config.example.toml for a starter template."`
}

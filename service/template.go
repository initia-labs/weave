package service

type Template string

// DarwinRunUpgradableCosmovisorTemplate should inject the arguments as follows: [1:binaryName, 2:binaryPath, 3:appHome, 4:userHome, 5:weaveLogPath, 6:serviceName]
const DarwinRunUpgradableCosmovisorTemplate Template = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.%[1]s.daemon</string>

    <key>ProgramArguments</key>
    <array>
        <string>%[2]s/%[1]s</string>
        <string>run</string>
        <string>start</string>
    </array>

    <key>RunAtLoad</key>
    <false/>

    <key>KeepAlive</key>
    <false/>

    <!-- Adding the environment variable -->
    <key>EnvironmentVariables</key>
    <dict>
		<key>HOME</key>
        <string>%[4]s</string>
        <key>DYLD_LIBRARY_PATH</key>
        <string>%[3]s/cosmovisor/dyld_lib</string>
        <key>DAEMON_NAME</key>
        <string>initiad</string>
        <key>DAEMON_HOME</key>
        <string>%[3]s</string>
        <key>DAEMON_ALLOW_DOWNLOAD_BINARIES</key>
        <string>true</string>
        <key>DAEMON_RESTART_AFTER_UPGRADE</key>
        <string>true</string>
    </dict>

    <key>StandardOutPath</key>
    <string>%[5]s/%[1]s.stdout.log</string>

    <key>StandardErrorPath</key>
    <string>%[5]s/%[1]s.stderr.log</string>

    <key>HardResourceLimits</key>
    <dict>
        <key>NumberOfFiles</key>
        <integer>65535</integer>
    </dict>
</dict>
</plist>
`

// DarwinRunNonUpgradableCosmovisorTemplate should inject the arguments as follows: [1:binaryName, 2:binaryPath, 3:appHome, 4:userHome, 5:weaveLogPath, 6:serviceName]
const DarwinRunNonUpgradableCosmovisorTemplate Template = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.%[1]s.daemon</string>

    <key>ProgramArguments</key>
    <array>
        <string>%[2]s/%[1]s</string>
        <string>run</string>
        <string>start</string>
    </array>

    <key>RunAtLoad</key>
    <false/>

    <key>KeepAlive</key>
    <false/>

    <!-- Adding the environment variable -->
    <key>EnvironmentVariables</key>
    <dict>
		<key>HOME</key>
        <string>%[4]s</string>
        <key>DYLD_LIBRARY_PATH</key>
        <string>%[3]s/cosmovisor/dyld_lib</string>
        <key>DAEMON_NAME</key>
        <string>initiad</string>
        <key>DAEMON_HOME</key>
        <string>%[3]s</string>
        <key>DAEMON_ALLOW_DOWNLOAD_BINARIES</key>
        <string>false</string>
        <key>DAEMON_RESTART_AFTER_UPGRADE</key>
        <string>false</string>
    </dict>

    <key>StandardOutPath</key>
    <string>%[5]s/%[1]s.stdout.log</string>

    <key>StandardErrorPath</key>
    <string>%[5]s/%[1]s.stderr.log</string>

    <key>HardResourceLimits</key>
    <dict>
        <key>NumberOfFiles</key>
        <integer>65535</integer>
    </dict>
</dict>
</plist>
`

// DarwinRunBinaryTemplate should inject the arguments as follows: [1:binaryName, 2:binaryPath, 3:appHome, 4:userHome, 5:weaveLogPath, 6:serviceName]
const DarwinRunBinaryTemplate Template = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.%[1]s.daemon</string>

    <key>ProgramArguments</key>
    <array>
        <string>%[2]s/%[1]s</string>
        <string>start</string>
        <string>--home=%[3]s</string>
    </array>

    <key>RunAtLoad</key>
    <false/>

    <key>KeepAlive</key>
    <false/>

    <!-- Adding the environment variable -->
    <key>EnvironmentVariables</key>
    <dict>
		<key>HOME</key>
        <string>%[4]s</string>
        <key>DYLD_LIBRARY_PATH</key>
        <string>%[2]s</string>
    </dict>

    <key>StandardOutPath</key>
    <string>%[5]s/%[1]s.stdout.log</string>

    <key>StandardErrorPath</key>
    <string>%[5]s/%[1]s.stderr.log</string>

    <key>HardResourceLimits</key>
    <dict>
        <key>NumberOfFiles</key>
        <integer>65535</integer>
    </dict>
</dict>
</plist>
`

// DarwinOPinitBotTemplate should inject the arguments as follows: [binaryName, binaryPath, appHome, userHome, weaveLogPath, serviceName]
const DarwinOPinitBotTemplate Template = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.%[1]s.%[6]s.daemon</string>

    <key>ProgramArguments</key>
    <array>
        <string>%[2]s/%[1]s</string>
        <string>start</string>
		<string>%[6]s</string>
		<string>--home=%[3]s</string>
    </array>

    <key>RunAtLoad</key>
    <false/>

    <key>KeepAlive</key>
    <false/>

    <!-- Adding the environment variable -->
    <key>EnvironmentVariables</key>
    <dict>
		<key>HOME</key>
        <string>%[4]s</string>
    </dict>

    <key>StandardOutPath</key>
    <string>%[5]s/%[1]s.%[6]s.stdout.log</string>

    <key>StandardErrorPath</key>
    <string>%[5]s/%[1]s.%[6]s.stderr.log</string>

    <key>HardResourceLimits</key>
    <dict>
        <key>NumberOfFiles</key>
        <integer>65535</integer>
    </dict>
</dict>
</plist>
`

// DarwinRelayerTemplate should inject the arguments as follows: [binaryName, binaryPath, appHome, userHome, weaveLogPath, serviceName]
const DarwinRelayerTemplate Template = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.%[1]s.daemon</string>

    <key>ProgramArguments</key>
    <array>
        <string>%[2]s/%[1]s</string>
        <string>start</string>
    </array>

    <key>RunAtLoad</key>
    <false/>

    <key>KeepAlive</key>
    <false/>

    <!-- Adding the environment variable -->
    <key>EnvironmentVariables</key>
    <dict>
		<key>HOME</key>
        <string>%[4]s</string>
    </dict>

    <key>StandardOutPath</key>
    <string>%[5]s/%[1]s.stdout.log</string>

    <key>StandardErrorPath</key>
    <string>%[5]s/%[1]s.stderr.log</string>

    <key>HardResourceLimits</key>
    <dict>
        <key>NumberOfFiles</key>
        <integer>65535</integer>
    </dict>
</dict>
</plist>
`

// LinuxRunUpgradableCosmovisorTemplate should inject the arguments as follows: [binaryName, binaryPath, serviceName, appHome, UserField]
const LinuxRunUpgradableCosmovisorTemplate Template = `
[Unit]
Description=%[1]s
After=network.target

[Service]
Type=exec
%[5]sExecStart=%[2]s/%[1]s run start
KillSignal=SIGINT
Environment="LD_LIBRARY_PATH=%[4]s/cosmovisor/dyld_lib"
Environment="DAEMON_NAME=initiad"
Environment="DAEMON_HOME=%[4]s"
Environment="DAEMON_ALLOW_DOWNLOAD_BINARIES=true"
Environment="DAEMON_RESTART_AFTER_UPGRADE=true"
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
`

// LinuxRunNonUpgradableCosmovisorTemplate should inject the arguments as follows: [binaryName, binaryPath, serviceName, appHome]
const LinuxRunNonUpgradableCosmovisorTemplate Template = `
[Unit]
Description=%[1]s
After=network.target

[Service]
Type=exec
%[5]sExecStart=%[2]s/%[1]s run start
KillSignal=SIGINT
Environment="LD_LIBRARY_PATH=%[4]s/cosmovisor/dyld_lib"
Environment="DAEMON_NAME=initiad"
Environment="DAEMON_HOME=%[4]s"
Environment="DAEMON_ALLOW_DOWNLOAD_BINARIES=false"
Environment="DAEMON_RESTART_AFTER_UPGRADE=false"
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
`

// LinuxRunMinitiaBinaryTemplate should inject the arguments as follows: [binaryName, binaryPath, serviceName, appHome]
const LinuxRunMinitiaBinaryTemplate Template = `
[Unit]
Description=%[1]s
After=network.target

[Service]
Type=exec
%[5]sExecStart=%[2]s/%[1]s start --home %[4]s
KillSignal=SIGINT
Environment="LD_LIBRARY_PATH=%[2]s"
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
`

// LinuxOPinitBotTemplate should inject the arguments as follows: [binaryName, binaryPath, serviceName, appHome]
const LinuxOPinitBotTemplate Template = `
[Unit]
Description=%[1]s %[3]s
After=network.target

[Service]
Type=exec
%[5]sExecStart=%[2]s/%[1]s start %[3]s --home %[4]s
KillSignal=SIGINT
Environment="LD_LIBRARY_PATH=%[2]s"
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
`

// LinuxRelayerTemplate should inject the arguments as follows: [binaryName, binaryPath, serviceName, appHome]
const LinuxRelayerTemplate Template = `
[Unit]
Description=%[1]s
After=network.target

[Service]
Type=exec
%[5]sExecStart=%[2]s/%[1]s start
KillSignal=SIGINT
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
`

var (
	LinuxTemplateMap = map[CommandName]Template{
		UpgradableInitia:    LinuxRunUpgradableCosmovisorTemplate,
		NonUpgradableInitia: LinuxRunNonUpgradableCosmovisorTemplate,
		Minitia:             LinuxRunMinitiaBinaryTemplate,
		OPinitExecutor:      LinuxOPinitBotTemplate,
		OPinitChallenger:    LinuxOPinitBotTemplate,
		Relayer:             LinuxRelayerTemplate,
	}
	DarwinTemplateMap = map[CommandName]Template{
		UpgradableInitia:    DarwinRunUpgradableCosmovisorTemplate,
		NonUpgradableInitia: DarwinRunNonUpgradableCosmovisorTemplate,
		Minitia:             DarwinRunBinaryTemplate,
		OPinitExecutor:      DarwinOPinitBotTemplate,
		OPinitChallenger:    DarwinOPinitBotTemplate,
		Relayer:             DarwinRelayerTemplate,
	}
)

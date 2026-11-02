package updatecheck

// Where this build looks for its own updates. This is a fork, so it points at
// the fork's GitHub releases: left on the official repository, the updater
// would quietly replace this build with the official one and drop every change
// made here.
const (
	// UpdateRepository is the "owner/name" GitHub repository releases come from.
	UpdateRepository = "DestroyersTaken/TcNo-Acc-Switcher"

	// ReleasesPageURL is opened when the user asks to download an update.
	ReleasesPageURL = "https://github.com/" + UpdateRepository + "/releases/latest"

	// UseTcnoAPIFallback controls the tcno.co version check that runs when the
	// GitHub check fails. It compares against the official app's version, so in
	// a fork it would keep announcing official releases. Off here.
	UseTcnoAPIFallback = false
)

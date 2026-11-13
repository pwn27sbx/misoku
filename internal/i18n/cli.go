package i18n

// Err is an error that reads key in the language in use each time it is
// read, for a sentinel error made before the language is known (a
// package variable). Each call makes a distinct error, so errors.Is
// matches the variable it is kept in and nothing else.
func Err(key Key) error { return &keyError{key: key} }

type keyError struct{ key Key }

func (e *keyError) Error() string { return T(e.key) }

// The command line: usage, flags and the messages of misoku and its
// subcommands (cmd/misoku).
const (
	CliUsage             Key = "cli.usage"
	CliFlagDir           Key = "cli.flag.dir"
	CliFlagVersion       Key = "cli.flag.version"
	CliFlagTheme         Key = "cli.flag.theme"
	CliFlagThemesDir     Key = "cli.flag.themes_dir"
	CliFlagListThemes    Key = "cli.flag.list_themes"
	CliIgnoringConfig    Key = "cli.ignoring_config"
	CliIgnoring          Key = "cli.ignoring"
	CliThemesPathMissing Key = "cli.themes_path_missing"
	CliUnknownTheme      Key = "cli.unknown_theme"
	CliOpenFailed        Key = "cli.open_failed"
	CliLogFailed         Key = "cli.log_failed"
	CliRunFailed         Key = "cli.run_failed"
	CliNotesMissing      Key = "cli.notes_missing"

	CliExportUsage   Key = "cli.export.usage"
	CliExportFlagOut Key = "cli.export.flag_out"
	CliReadFailed    Key = "cli.export.read_failed"
	CliExportFailed  Key = "cli.export.failed"

	CliPasteUsage    Key = "cli.paste.usage"
	CliPasteFlagDir  Key = "cli.paste.flag_dir"
	CliPasteFlagLink Key = "cli.paste.flag_link"
	CliPasteNotDir   Key = "cli.paste.not_dir"
	CliPasteNeeds    Key = "cli.paste.needs"
	CliPasteNoImage  Key = "cli.paste.no_image"
	CliPasteMkdir    Key = "cli.paste.mkdir"
	CliPasteNoName   Key = "cli.paste.no_name"
	CliPasteWrite    Key = "cli.paste.write"

	CliTutorUsage       Key = "cli.tutor.usage"
	CliTutorFlagReset   Key = "cli.tutor.flag_reset"
	CliTutorFlagRoute   Key = "cli.tutor.flag_route"
	CliTutorUnexpected  Key = "cli.tutor.unexpected"
	CliTutorRouteNeeds  Key = "cli.tutor.route_needs"
	CliTutorRouteBad    Key = "cli.tutor.route_bad"
	CliTutorGitFailed   Key = "cli.tutor.git_failed"
	CliTutorBadProgress Key = "cli.tutor.bad_progress"

	CliSyncInitUsage   Key = "cli.sync.init_usage"
	CliSyncSetupUsage  Key = "cli.sync.setup_usage"
	CliSyncSetupLine   Key = "cli.sync.setup_line"
	CliSyncFlagDir     Key = "cli.sync.flag_dir"
	CliInitBadConfig   Key = "cli.sync.init_bad_config"
	CliInitSettingUp   Key = "cli.sync.init_setting_up"
	CliInitUnencrypted Key = "cli.sync.init_unencrypted"
	CliInitNothingLost Key = "cli.sync.init_nothing_lost"
	CliInitNotListed   Key = "cli.sync.init_not_listed"
	CliInitAlready     Key = "cli.sync.init_already"
	CliInitDone        Key = "cli.sync.init_done"
	CliSetupSettingUp  Key = "cli.sync.setup_setting_up"
	CliSetupStart      Key = "cli.sync.setup_start"
	CliSetupName       Key = "cli.sync.setup_name"
	CliSetupEmail      Key = "cli.sync.setup_email"
	CliSetupSignInNow  Key = "cli.sync.setup_sign_in_now"
	CliSetupRepoName   Key = "cli.sync.setup_repo_name"
	CliSetupPickRepo   Key = "cli.sync.setup_pick_repo"
	CliSetupMatching   Key = "cli.sync.setup_matching"
	CliSetupPrivate    Key = "cli.sync.setup_private"
	CliSetupNotAllowed Key = "cli.sync.setup_not_allowed"
	// CliYesNo and CliYesDefault are the hints after a yes/no question;
	// CliYesWords the answers that mean yes, space-separated.
	CliYesNo      Key = "cli.yes_no"
	CliYesDefault Key = "cli.yes_default"
	CliYesWords   Key = "cli.yes_words"
)

// The welcome before the app, in the terminal and on its screen.
const (
	WelcomeTitle         Key = "welcome.title"
	WelcomeMissing       Key = "welcome.missing"
	WelcomeNotesQuestion Key = "welcome.notes_question"
	WelcomeStart         Key = "welcome.start"
	WelcomeBring         Key = "welcome.bring"
	WelcomeUseThis       Key = "welcome.use_this"
	WelcomeNumber        Key = "welcome.number"
	WelcomeType          Key = "welcome.type"
	WelcomeOneOrTwo      Key = "welcome.one_or_two"
	WelcomeOneTwoThree   Key = "welcome.one_two_three"
	WelcomeStopped       Key = "welcome.stopped"
	WelcomeFolderAsk     Key = "welcome.folder_ask"
	WelcomeFolder        Key = "welcome.folder"
	WelcomeFolderShown   Key = "welcome.folder_shown"
	WelcomeFullPath      Key = "welcome.full_path"
	WelcomeUnsafe        Key = "welcome.unsafe"
	WelcomeIsFile        Key = "welcome.is_file"
	WelcomeNoParent      Key = "welcome.no_parent"
	WelcomeCreateFailed  Key = "welcome.create_failed"
	WelcomeNotRemembered Key = "welcome.not_remembered"
	WelcomeShowFailed    Key = "welcome.show_failed"
	WelcomeHintChoose    Key = "welcome.hint_choose"
	WelcomeHintCancel    Key = "welcome.hint_cancel"
	WelcomeHintCreate    Key = "welcome.hint_create"
	WelcomeHintBring     Key = "welcome.hint_bring"
)

// The sync setup assistant (gitsync's Setup), in every front end.
const (
	SetupBadConfig        Key = "setup.bad_config"
	SetupBringTitle       Key = "setup.bring_title"
	SetupBringLine        Key = "setup.bring_line"
	SetupSyncTitle        Key = "setup.sync_title"
	SetupSyncLine         Key = "setup.sync_line"
	SetupPrivateLine      Key = "setup.private_line"
	SetupGHAccess         Key = "setup.gh_access"
	SetupCreateChoice     Key = "setup.create_choice"
	SetupFreshChoice      Key = "setup.fresh_choice"
	SetupNoParent         Key = "setup.no_parent"
	SetupNotFolder        Key = "setup.not_folder"
	SetupAlreadySynced    Key = "setup.already_synced"
	SetupPausedThough     Key = "setup.paused_though"
	SetupStatusHint       Key = "setup.status_hint"
	SetupTitle            Key = "setup.title"
	SetupAlreadyRepo      Key = "setup.already_repo"
	SetupAlreadyRepoHint  Key = "setup.already_repo_hint"
	SetupOwnRepoTitle     Key = "setup.own_repo_title"
	SetupOwnRepoLine      Key = "setup.own_repo_line"
	SetupOwnRepoQuestion  Key = "setup.own_repo_question"
	SetupLookFailed       Key = "setup.look_failed"
	SetupRemoveFailed     Key = "setup.remove_failed"
	SetupChangedNothing   Key = "setup.changed_nothing"
	SetupCancelled        Key = "setup.cancelled"
	SetupEnded            Key = "setup.ended"
	SetupInitFailed       Key = "setup.init_failed"
	SetupMadeOwnRepo      Key = "setup.made_own_repo"
	SetupMadeOwnRepoNote  Key = "setup.made_own_repo_note"
	SetupMadeFolder       Key = "setup.made_folder"
	SetupMadeRepo         Key = "setup.made_repo"
	SetupLeftNotEmpty     Key = "setup.left_not_empty"
	SetupNotRemoved       Key = "setup.not_removed"
	SetupFolderRemoved    Key = "setup.folder_removed"
	SetupLeftUnknown      Key = "setup.left_unknown"
	SetupLeftCommits      Key = "setup.left_commits"
	SetupLeftNotOurs      Key = "setup.left_not_ours"
	SetupRepoNotRemoved   Key = "setup.repo_not_removed"
	SetupRepoRemoved      Key = "setup.repo_removed"
	SetupAppleGit         Key = "setup.apple_git"
	SetupGitDidNotRun     Key = "setup.git_did_not_run"
	SetupGitTooOld        Key = "setup.git_too_old"
	SetupNoGH             Key = "setup.no_gh"
	SetupIdentityLine     Key = "setup.identity_line"
	SetupIdentityGitHub   Key = "setup.identity_github"
	SetupIdentityTitle    Key = "setup.identity_title"
	SetupIdentitySave     Key = "setup.identity_save"
	SetupTypeName         Key = "setup.type_name"
	SetupTypeEmail        Key = "setup.type_email"
	SetupNameNotSaved     Key = "setup.name_not_saved"
	SetupEmailNotSaved    Key = "setup.email_not_saved"
	SetupNameSaved        Key = "setup.name_saved"
	SetupIdentitySaved    Key = "setup.identity_saved"
	SetupKnowsYou         Key = "setup.knows_you"
	SetupSignInLine       Key = "setup.sign_in_line"
	SetupSignInKeychain   Key = "setup.sign_in_keychain"
	SetupNoBrowser        Key = "setup.no_browser"
	SetupSignInTitle      Key = "setup.sign_in_title"
	SetupSignedIn         Key = "setup.signed_in"
	SetupInstallMac       Key = "setup.install_mac"
	SetupInstallWindows   Key = "setup.install_windows"
	SetupInstallLinux     Key = "setup.install_linux"
	SetupNotSignedIn      Key = "setup.not_signed_in"
	SetupListFailed       Key = "setup.list_failed"
	SetupListUnreadable   Key = "setup.list_unreadable"
	SetupWhichBring       Key = "setup.which_bring"
	SetupPickBring        Key = "setup.pick_bring"
	SetupWhich            Key = "setup.which"
	SetupPick             Key = "setup.pick"
	SetupNotYours         Key = "setup.not_yours"
	SetupNoNotes          Key = "setup.no_notes"
	SetupStartTitle       Key = "setup.start_title"
	SetupNewTitle         Key = "setup.new_title"
	SetupNewLine          Key = "setup.new_line"
	SetupChooseListed     Key = "setup.choose_listed"
	SetupPublic           Key = "setup.public"
	SetupNameRule         Key = "setup.name_rule"
	SetupNameYours        Key = "setup.name_yours"
	SetupNameTaken        Key = "setup.name_taken"
	SetupCreateFailed     Key = "setup.create_failed"
	SetupCreatedOnGitHub  Key = "setup.created_on_github"
	SetupCreatedNowhere   Key = "setup.created_nowhere"
	SetupGitAccessFailed  Key = "setup.git_access_failed"
	SetupGitAccess        Key = "setup.git_access"
	SetupCreated          Key = "setup.created"
	SetupFixAndPick       Key = "setup.fix_and_pick"
	SetupNothingLost      Key = "setup.nothing_lost"
	SetupNotListed        Key = "setup.not_listed"
	SetupWhileOpen        Key = "setup.while_open"
	SetupEvery            Key = "setup.every"
	SetupDone             Key = "setup.done"
	SetupOtherComputer    Key = "setup.other_computer"
	SetupDoneTitle        Key = "setup.done_title"
	SetupDidNotFinish     Key = "setup.did_not_finish"
	SetupNothingChanged   Key = "setup.nothing_changed"
	SetupStopped          Key = "setup.stopped"
	SetupStopNothing      Key = "setup.stop_nothing"
	SetupStopChanged      Key = "setup.stop_changed"
	SetupChanges          Key = "setup.changes"
	SetupStoppedTitle     Key = "setup.stopped_title"
	SetupNoAnswer         Key = "setup.no_answer"
	SetupSignInUnfinished Key = "setup.sign_in_unfinished"
	SetupReadFailed       Key = "setup.read_failed"
	SetupNothingFetched   Key = "setup.nothing_fetched"
	SetupCodeProject      Key = "setup.code_project"
	SetupAndMore          Key = "setup.and_more"
	SetupUnrelated        Key = "setup.unrelated"
	SetupNotNotes         Key = "setup.not_notes"
	SetupMixLine          Key = "setup.mix_line"
	SetupMixCancel        Key = "setup.mix_cancel"
	SetupMixTitle         Key = "setup.mix_title"
	SetupMixQuestion      Key = "setup.mix_question"
	SetupOtherFiles       Key = "setup.other_files"
)

// The messages of sync itself (gitsync): why it pauses or stops, and what
// `misoku sync init` did.
const (
	GitOff              Key = "gitsync.off"
	GitNotListed        Key = "gitsync.not_listed"
	GitMissing          Key = "gitsync.missing"
	GitUnknownVersion   Key = "gitsync.unknown_version"
	GitTooOld           Key = "gitsync.too_old"
	GitCheckFailed      Key = "gitsync.check_failed"
	GitNotARepoHint     Key = "gitsync.not_a_repo_hint"
	GitNotRepoRoot      Key = "gitsync.not_repo_root"
	GitNoRemote         Key = "gitsync.no_remote"
	GitInProgress       Key = "gitsync.in_progress"
	GitOpMerge          Key = "gitsync.op.merge"
	GitOpRebase         Key = "gitsync.op.rebase"
	GitOpCherryPick     Key = "gitsync.op.cherry_pick"
	GitOpRevert         Key = "gitsync.op.revert"
	GitDetached         Key = "gitsync.detached"
	GitNoIdentity       Key = "gitsync.no_identity"
	GitLocked           Key = "gitsync.locked"
	GitLockChanged      Key = "gitsync.lock_changed"
	GitStillRunning     Key = "gitsync.still_running"
	GitEarlierRunning   Key = "gitsync.earlier_running"
	GitNotARepo         Key = "gitsync.not_a_repo"
	GitNotInstalled     Key = "gitsync.not_installed"
	GitHostKeyChanged   Key = "gitsync.host_key_changed"
	GitHostKeyUnknown   Key = "gitsync.host_key_unknown"
	GitHostKeyRefused   Key = "gitsync.host_key_refused"
	GitKeyRefused       Key = "gitsync.key_refused"
	GitKeyRefusedInit   Key = "gitsync.key_refused_init"
	GitSignInFailed     Key = "gitsync.sign_in_failed"
	GitSignInFailedInit Key = "gitsync.sign_in_failed_init"
	GitNoReason         Key = "gitsync.no_reason"
	GitCouldNotFetch    Key = "gitsync.could_not_fetch"
	GitCouldNotPush     Key = "gitsync.could_not_push"
	GitCouldNotReach    Key = "gitsync.could_not_reach"
	GitNeedsLock        Key = "gitsync.needs_lock"
	GitCountFailed      Key = "gitsync.count_failed"
	GitPushTurnedDown   Key = "gitsync.push_turned_down"
	GitSyncStopped      Key = "gitsync.sync_stopped"
	GitPushedFirst      Key = "gitsync.pushed_first"
	GitPushRefused      Key = "gitsync.push_refused"
	GitRewritten        Key = "gitsync.rewritten"
	GitUnrelated        Key = "gitsync.unrelated"
	GitNoDefaultBranch  Key = "gitsync.no_default_branch"
	GitFileBusy         Key = "gitsync.file_busy"
	GitApplyFailed      Key = "gitsync.apply_failed"
	GitBigClash         Key = "gitsync.big_clash"
	GitNoticeClash      Key = "gitsync.notice_clash"
	GitFileChanged      Key = "gitsync.file_changed"
	GitNoAnswerAfter    Key = "gitsync.no_answer_after"
	GitCancelled        Key = "gitsync.cancelled"
	GitCallerTimeout    Key = "gitsync.caller_timeout"
	GitNoGitDir         Key = "gitsync.no_git_dir"
	GitUnsafeFolder     Key = "gitsync.unsafe_folder"
	GitAnotherWindow    Key = "gitsync.another_window"
	GitRemoveOrigin     Key = "gitsync.remove_origin"
	GitOtherOrigin      Key = "gitsync.other_origin"
	GitThenInitAgain    Key = "gitsync.then_init_again"
	GitTracksFiles      Key = "gitsync.tracks_files"
	GitSubmoduleOf      Key = "gitsync.submodule_of"
	GitTwoRepos         Key = "gitsync.two_repos"
	GitInsideRepo       Key = "gitsync.inside_repo"
	GitNotAFolder       Key = "gitsync.not_a_folder"
	GitNoParent         Key = "gitsync.no_parent"
	GitInitDidNotRun    Key = "gitsync.init_did_not_run"
	GitInitTooOld       Key = "gitsync.init_too_old"
	GitReportRepo       Key = "gitsync.report.repo"
	GitReportOrigin     Key = "gitsync.report.origin"
	GitReportAttrs      Key = "gitsync.report.attrs"
	GitReportKeptAttrs  Key = "gitsync.report.kept_attrs"
	GitReportLeftOut    Key = "gitsync.report.left_out"
	GitReportCommitted  Key = "gitsync.report.committed"
	GitReportBrought    Key = "gitsync.report.brought"
	GitReportConflicts  Key = "gitsync.report.conflicts"
	GitReportResolve    Key = "gitsync.report.resolve"
	GitReportPushed     Key = "gitsync.report.pushed"
	GitCreatedFolder    Key = "gitsync.created_folder"
	GitAddedToVaults    Key = "gitsync.added_to_vaults"
	GitNoticeNested     Key = "gitsync.notice.nested"
	GitNoticeSubmodule  Key = "gitsync.notice.submodule"
	GitNoticeLink       Key = "gitsync.notice.link"
	GitNoticeWinLeftOut Key = "gitsync.notice.win_left_out"
	GitNoticeWinName    Key = "gitsync.notice.win_name"
	GitNoticeCaseClash  Key = "gitsync.notice.case_clash"
	GitNoticeCaseDup    Key = "gitsync.notice.case_dup"
	GitNoticeHeld       Key = "gitsync.notice.held"
	GitNameControl      Key = "gitsync.name.control"
	GitNameDotSpace     Key = "gitsync.name.dot_space"
	GitNameReserved     Key = "gitsync.name.reserved"
)

// The errors of the notes store (core) the bar shows.
const (
	CoreNotFound          Key = "core.not_found"
	CoreNoTrash           Key = "core.no_trash"
	CoreBadWorkspace      Key = "core.bad_workspace"
	CoreMkWorkspace       Key = "core.mk_workspace"
	CoreWorkspaceAccess   Key = "core.workspace_access"
	CoreNotADir           Key = "core.not_a_dir"
	CoreNameEmpty         Key = "core.name_empty"
	CoreNameChars         Key = "core.name_chars"
	CoreNameReserved      Key = "core.name_reserved"
	CoreOutside           Key = "core.outside"
	CoreHidden            Key = "core.hidden"
	CoreExists            Key = "core.exists"
	CoreMkFolder          Key = "core.mk_folder"
	CoreSaveFailed        Key = "core.save_failed"
	CoreLinksFailed       Key = "core.links_failed"
	CoreIsFolder          Key = "core.is_folder"
	CoreMkAssets          Key = "core.mk_assets"
	CoreCopyFailed        Key = "core.copy_failed"
	CoreMoveRoot          Key = "core.move_root"
	CoreMoveInto          Key = "core.move_into"
	CoreAlreadyThere      Key = "core.already_there"
	CoreExistsThere       Key = "core.exists_there"
	CoreMoveFolder        Key = "core.move_folder"
	CoreMovePage          Key = "core.move_page"
	CoreMoveSubNothing    Key = "core.move_sub_nothing"
	CoreMoveSubHalf       Key = "core.move_sub_half"
	CoreMoveReadBack      Key = "core.move_read_back"
	CoreBackslash         Key = "core.backslash"
	CoreTrashMissing      Key = "core.trash_missing"
	CoreTrashFailed       Key = "core.trash_failed"
	CoreTrashFull         Key = "core.trash_full"
	CoreCreateFailed      Key = "core.create_failed"
	CoreRelativeFolder    Key = "core.relative_folder"
	CoreRelativeWorkspace Key = "core.relative_workspace"
	CoreEmptySegment      Key = "core.empty_segment"
	CoreDots              Key = "core.dots"
	CorePageExists        Key = "core.page_exists"
	CoreFolderExists      Key = "core.folder_exists"
	CoreRenameFailed      Key = "core.rename_failed"
	CoreRenameSubBack     Key = "core.rename_sub_back"
	CoreRenameSubNothing  Key = "core.rename_sub_nothing"
	CoreRenameFolder      Key = "core.rename_folder"
	CoreDeleteFolder      Key = "core.delete_folder"
	CoreDeletePage        Key = "core.delete_page"
	CoreDeleteSubHalf     Key = "core.delete_sub_half"
)

// The graphs: the local one around a note (graph) and the vault map
// (globalgraph).
const (
	GraphNoNotes          Key = "graph.no_notes"
	GraphTooSmall         Key = "graph.too_small"
	GraphAllNotes         Key = "graph.all_notes"
	GraphLinks            Key = "graph.links"
	GraphBacklinks        Key = "graph.backlinks"
	GraphNotes            Key = "graph.notes"
	GraphSteps            Key = "graph.steps"
	GraphOrphans          Key = "graph.orphans"
	GraphUnresolved       Key = "graph.unresolved"
	GraphMatches          Key = "graph.matches"
	GraphOrphansHidden    Key = "graph.orphans_hidden"
	GraphUnresolvedHidden Key = "graph.unresolved_hidden"
	GraphFilterOn         Key = "graph.filter_on"
	GraphCentre           Key = "graph.centre"
	GraphBack             Key = "graph.back"
	GraphReset            Key = "graph.reset"
	GraphFilter           Key = "graph.filter"
	GraphOrphansBtn       Key = "graph.orphans_button"
	GraphUnresBtn         Key = "graph.unresolved_button"
	GraphTailFull         Key = "graph.tail_full"
	GraphTailDrag         Key = "graph.tail_drag"
	GraphTailOpen         Key = "graph.tail_open"
	GraphTailKeys         Key = "graph.tail_keys"
	GraphMapTailFull      Key = "graph.map_tail_full"
	GraphMapTailDrag      Key = "graph.map_tail_drag"
	GraphMapTailOpen      Key = "graph.map_tail_open"
	GraphPromptJump       Key = "graph.prompt_jump"
)

func init() {
	register(EN, map[Key]string{
		CliUsage: `Usage:
  misoku [folder] [flags]                  open your notes: the folder given, else $MISOKU_WORKSPACE,
                                           else notes_dir in the config, else the current folder
  misoku export <note.md> [-o out.pdf]     export a note to PDF
  misoku paste-image [--dir <folder>]      save the picture in the clipboard under <folder>/assets
                                           and print the path to link it (--link prints the link)
  misoku sync setup [--dir <folder>]       sync a folder through a private GitHub repository (recommended)
  misoku sync init <url> [--dir <folder>]  set up git sync of a folder with any git URL (SSH, other services)
  misoku tutor [--route B|I|A|E] [--reset]  learn Misoku and Markdown by routes (--reset starts over)

Flags:
`,
		CliFlagDir:           "folder to open, like misoku <folder> (defaults to $MISOKU_WORKSPACE, then notes_dir in the config, then the current folder)",
		CliFlagVersion:       "print the Misoku version and exit",
		CliFlagTheme:         "colour theme for this session (see --list-themes)",
		CliFlagThemesDir:     "directory or .json file with custom themes (overrides config themes_dir)",
		CliFlagListThemes:    "list the available colour themes and exit",
		CliIgnoringConfig:    "Ignoring unreadable config %s: %v",
		CliIgnoring:          "Ignoring %v",
		CliThemesPathMissing: "Custom themes path %s not found; using bundled themes only",
		CliUnknownTheme:      "Unknown theme %q (run misoku --list-themes)",
		CliOpenFailed:        "Could not open workspace %q: %v",
		CliLogFailed:         "Could not open log file: %v",
		CliRunFailed:         "Error running Misoku application: %v",
		CliNotesMissing:      "Your notes folder %s is missing; open another with misoku <folder>",

		CliExportUsage:   "Usage: misoku export <note.md> [-o out.pdf]",
		CliExportFlagOut: "output file or folder (defaults to the note's name with .pdf, next to it)",
		CliReadFailed:    "Could not read %s: %v",
		CliExportFailed:  "Export failed: %v",

		CliPasteUsage:    "Usage: misoku paste-image [--dir <folder>] [--link]",
		CliPasteFlagDir:  "folder of the note the picture belongs to",
		CliPasteFlagLink: "print the Markdown link instead of the path",
		CliPasteNotDir:   "%s is not a folder",
		CliPasteNeeds:    "needs %s",
		CliPasteNoImage:  "the clipboard holds no picture",
		CliPasteMkdir:    "could not create %s: %w",
		CliPasteNoName:   "could not find a free name for %s",
		CliPasteWrite:    "could not write %s: %w",

		CliTutorUsage:       "Usage: misoku tutor [--route B|I|A|E] [--reset]",
		CliTutorFlagReset:   "start the tutorial over from the bundled lessons",
		CliTutorFlagRoute:   "route to open (B, I, A or E)",
		CliTutorUnexpected:  "unexpected argument %q",
		CliTutorRouteNeeds:  "--route needs B, I, A or E",
		CliTutorRouteBad:    "unknown route %q (want B, I, A or E)",
		CliTutorGitFailed:   "could not prepare the tutorial's git history: %v",
		CliTutorBadProgress: "ignoring the progress file: %v",

		CliSyncInitUsage:   "Usage: misoku sync init <url> [--dir <folder>]",
		CliSyncSetupUsage:  "Usage: misoku sync setup [--dir <folder>]",
		CliSyncSetupLine:   "       misoku sync setup [--dir <folder>]   (recommended: connects a private GitHub repository for you)",
		CliSyncFlagDir:     "vault folder (defaults to $MISOKU_WORKSPACE, then the current directory)",
		CliInitBadConfig:   "the config file %s could not be read (%v); fix it, then run init again",
		CliInitSettingUp:   "Setting up sync of %s with %s…",
		CliInitUnencrypted: "Your notes are stored unencrypted in that repository: keep it private.",
		CliInitNothingLost: "%v\nNothing was lost; fix that, then run `misoku sync init <url>` again.",
		CliInitNotListed:   "the vault is set up, but it could not be listed in the config (%v); add it to sync.vaults by hand:\n  {\"sync\": {\"vaults\": {%s: {}}}}",
		CliInitAlready:     "Already set up: this folder syncs with %s (branch %s); nothing changed.",
		CliInitDone:        "Done: Misoku syncs this folder when it opens it, and with :sync.",
		CliSetupSettingUp:  "Setting up sync of %s with GitHub.",
		CliSetupStart:      "Start?",
		CliSetupName:       "Your name",
		CliSetupEmail:      "Your email",
		CliSetupSignInNow:  "Sign in to GitHub now?",
		CliSetupRepoName:   "Repository name",
		CliSetupPickRepo:   "Number, name or part of one (Enter cancels): ",
		CliSetupMatching:   "Repositories matching %q:",
		CliSetupPrivate:    "private",
		CliSetupNotAllowed: "%s: not allowed, notes are stored unencrypted",
		CliYesNo:           " [y/N] ",
		CliYesDefault:      " [Y/n] ",
		CliYesWords:        "y yes",

		WelcomeTitle:         "Welcome to Misoku.",
		WelcomeMissing:       "I can't find your notes at %s.",
		WelcomeNotesQuestion: "Where do your notes live?",
		WelcomeStart:         "Start my notes",
		WelcomeBring:         "Bring my notes from another computer",
		WelcomeUseThis:       "Use this folder: %s",
		WelcomeNumber:        "Number (Enter cancels): ",
		WelcomeType:          "! Type %s.",
		WelcomeOneOrTwo:      "1 or 2",
		WelcomeOneTwoThree:   "1, 2 or 3",
		WelcomeStopped:       "Nothing was changed. Run misoku again when you are ready.",
		WelcomeFolderAsk:     "Folder for your notes [%s]: ",
		WelcomeFolder:        "Folder for your notes:",
		WelcomeFolderShown:   "Folder for your notes: %s",
		WelcomeFullPath:      "Type a full path, such as ~/Notes.",
		WelcomeUnsafe:        "%s is your home folder, the root of a disk or a system folder: choose a folder of its own for your notes, such as ~/Notes.",
		WelcomeIsFile:        "%s is a file, not a folder: choose another name",
		WelcomeNoParent:      "the folder %s does not exist: create it first, or choose another folder",
		WelcomeCreateFailed:  "could not create %s (%w)",
		WelcomeNotRemembered: "Could not remember %s as your notes folder (%v); misoku will ask again next time.",
		WelcomeShowFailed:    "Could not show the welcome (%v).",
		WelcomeHintChoose:    "↑↓ or a number · Enter choose · Esc or q leave",
		WelcomeHintCancel:    "Esc cancel",
		WelcomeHintCreate:    "Enter create and open · Esc leave",
		WelcomeHintBring:     "Enter bring the notes here · Esc leave",

		SetupBadConfig:        "The config file %s could not be read (%v); fix it, then run setup again.",
		SetupBringTitle:       "Bring your notes",
		SetupBringLine:        "Your notes come from the private GitHub repository your other computer syncs with, and stay in sync from then on.",
		SetupSyncTitle:        "Sync this vault",
		SetupSyncLine:         "Your notes will be kept in a private GitHub repository and reach your other computers on their own.",
		SetupPrivateLine:      "In the repository your notes are not encrypted, so it stays private.",
		SetupGHAccess:         "You need a GitHub account. Misoku signs in with gh, GitHub's command-line tool, which keeps the access: Misoku never sees or stores it.",
		SetupCreateChoice:     "Create a new private repository",
		SetupFreshChoice:      "I don't have notes on another computer → Start my notes",
		SetupNoParent:         "%s does not exist, nor does its folder %s; create it first, or choose another folder.",
		SetupNotFolder:        "%s is not a folder.",
		SetupAlreadySynced:    "This vault is already set up: it syncs with %s; nothing changed.",
		SetupPausedThough:     "Sync is paused, though: %s.",
		SetupStatusHint:       "`:sync status` in Misoku shows how it is going.",
		SetupTitle:            "Sync setup",
		SetupAlreadyRepo:      "This folder is already set up with the git repository %s, though Misoku does not sync it yet; nothing changed.",
		SetupAlreadyRepoHint:  "To sync it with that repository, run `misoku sync init %s`.",
		SetupOwnRepoTitle:     "A repository for your notes",
		SetupOwnRepoLine:      "This folder is inside the git repository at %s, which does not track it: syncing it as it is would commit that whole repository. A separate repository leaves that one as it is.",
		SetupOwnRepoQuestion:  "Create a separate repository for your notes here?",
		SetupLookFailed:       "Setup could not look at %s (%v). Setup changed nothing.",
		SetupRemoveFailed:     "Setup made the empty folder %s to look at it and could not remove it again (%v); delete it, then run setup again.",
		SetupChangedNothing:   "%s. Setup changed nothing.",
		SetupCancelled:        "Setup cancelled",
		SetupEnded:            "Setup has already ended; run it again.",
		SetupInitFailed:       "git could not create a repository in this folder (%v).",
		SetupMadeOwnRepo:      "A git repository of its own was created in this folder",
		SetupMadeOwnRepoNote:  "✓ created a git repository of its own for your notes",
		SetupMadeFolder:       "The folder %s was created",
		SetupMadeRepo:         "A git repository was created in %s",
		SetupLeftNotEmpty:     "; it was left in place, as it is not empty",
		SetupNotRemoved:       "; it could not be removed again (%v)",
		SetupFolderRemoved:    "The folder %s it created was removed",
		SetupLeftUnknown:      "; it was left in place, as git could not tell whether it has commits (%v)",
		SetupLeftCommits:      "; it already has commits, so it was left in place",
		SetupLeftNotOurs:      "; it was left in place, as %s is no longer the folder it created",
		SetupRepoNotRemoved:   "; it could not be removed again (%v): delete %s by hand",
		SetupRepoRemoved:      "The git repository it created was removed",
		SetupAppleGit:         "Apple is installing git; press Enter when it finishes.",
		SetupGitDidNotRun:     "git did not run (%v). Install git %s or newer, then run setup again.",
		SetupGitTooOld:        "git %s is too old: sync needs git %s or newer. Update git, then run setup again.",
		SetupNoGH:             "gh, GitHub's command-line tool, is not installed: setup uses it to sign in to GitHub.",
		SetupIdentityLine:     "git does not know your name and email yet: it writes them in every change it records (your notes' history).",
		SetupIdentityGitHub:   "Misoku filled them in from your GitHub account (the email is GitHub's private address for you); change them if you like.",
		SetupIdentityTitle:    "Your name for git",
		SetupIdentitySave:     "Save them in git's global settings (git config --global), for every repository on this computer?",
		SetupTypeName:         "Type your name (any name: it is only shown in the history).",
		SetupTypeEmail:        "Type an email such as you@example.com.",
		SetupNameNotSaved:     "git could not save user.name (%v).",
		SetupEmailNotSaved:    "git could not save user.email (%v).",
		SetupNameSaved:        "Your git name was saved",
		SetupIdentitySaved:    "Your git name and email were saved",
		SetupKnowsYou:         "✓ git knows you: %s <%s>",
		SetupSignInLine:       "Sign in to GitHub: gh shows a one-time code and opens your browser.",
		SetupSignInKeychain:   "Sign in there and enter the code. gh keeps the access in your system's keychain; Misoku never sees it.",
		SetupNoBrowser:        "This computer has no browser gh can open: when gh shows the address https://github.com/login/device, open it on any device (your phone works), sign in and enter the code.",
		SetupSignInTitle:      "Sign in to GitHub",
		SetupSignedIn:         "✓ signed in to GitHub",
		SetupInstallMac:       "Install it with `brew install gh` (see https://cli.github.com), then run setup again.",
		SetupInstallWindows:   "Install it with `winget install --id GitHub.cli` (see https://cli.github.com), then run setup again.",
		SetupInstallLinux:     "Install it with your package manager, for example `sudo apt install gh` (see https://cli.github.com), then run setup again.",
		SetupNotSignedIn:      "You are not signed in to GitHub (the sign-in did not finish). Run setup again to try once more.",
		SetupListFailed:       "gh could not list your repositories (%v).",
		SetupListUnreadable:   "gh listed your repositories in a way Misoku does not understand (%v).",
		SetupWhichBring:       "Which repository has your notes?",
		SetupPickBring:        "Pick the repository your other computer syncs with.",
		SetupWhich:            "Which repository keeps this vault?",
		SetupPick:             "Pick the repository your other computer already syncs with, or create a new private one.",
		SetupNotYours:         "%s is not one of your repositories.",
		SetupNoNotes:          "No notes to bring",
		SetupStartTitle:       "Start my notes",
		SetupNewTitle:         "Name of the new private repository",
		SetupNewLine:          "Misoku creates it private on GitHub and pushes your notes to it.",
		SetupChooseListed:     "Choose one of the listed repositories.",
		SetupPublic:           "%s is public: anyone could read your notes, which are stored unencrypted. Choose a private repository or create a new one (or make it private on GitHub first).",
		SetupNameRule:         "Use only letters, digits, '-', '_' and '.' (GitHub's rule).",
		SetupNameYours:        "You already have a repository named %s: pick it from the list (run setup again), or choose another name.",
		SetupNameTaken:        "A repository named %s already exists in your GitHub account; choose another name.",
		SetupCreateFailed:     "GitHub did not create the repository (%v).",
		SetupCreatedOnGitHub:  "the private repository %s was created on GitHub",
		SetupCreatedNowhere:   "GitHub created the repository %s, but gh did not say where it is. Run setup again and pick it from the list.",
		SetupGitAccessFailed:  "gh could not give git the access to GitHub (%v)",
		SetupGitAccess:        "GitHub access for git was set up with gh",
		SetupCreated:          "✓ Created the private repository %s.",
		SetupFixAndPick:       "The repository %s is on GitHub: fix that, then run setup again and pick it.",
		SetupNothingLost:      "%s; nothing was lost.",
		SetupNotListed:        "The vault is set up, but it could not be listed in the config (%v); add it to sync.vaults by hand: {\"sync\": {\"vaults\": {%s: {}}}}",
		SetupWhileOpen:        "while it is open",
		SetupEvery:            "every %s while it is open",
		SetupDone:             "Done: Misoku syncs this vault with %s %s, when it opens it and when you quit.",
		SetupOtherComputer:    "On your other computer: install Misoku, run misoku and choose \"%s\", then pick %s.",
		SetupDoneTitle:        "Sync is set up",
		SetupDidNotFinish:     "the sync did not finish",
		SetupNothingChanged:   "Nothing was changed.",
		SetupStopped:          "Setup stopped. %s",
		SetupStopNothing:      "%s; nothing was changed.",
		SetupStopChanged:      "%s. %s",
		SetupChanges:          "%s. Nothing else changed.",
		SetupStoppedTitle:     "Sync setup stopped",
		SetupNoAnswer:         "gh %s: no answer after %s",
		SetupSignInUnfinished: "the sign-in did not finish",
		SetupReadFailed:       "Misoku could not read %s to check it before using it (%s).",
		SetupNothingFetched:   "Nothing was fetched into this folder or pushed. Check your connection and your access to it, then run setup again.",
		SetupCodeProject:      "%s looks like a code project (%s): Misoku won't mix your notes into it. Pick another repository or create a new private one.",
		SetupAndMore:          ", and %d more",
		SetupUnrelated:        "%s does not share this folder's history. It holds: %s%s.",
		SetupNotNotes:         "%d of %d files are not notes",
		SetupMixLine:          "Using it puts both together: its files come into this folder, and your notes are pushed to it.",
		SetupMixCancel:        "If it is not where your notes belong, cancel and create a new private repository.",
		SetupMixTitle:         "This repository already has other files",
		SetupMixQuestion:      "Mix this folder's notes with %s?",
		SetupOtherFiles:       "It may not be a notes repository: %s.",

		GitOff:              "sync is turned off by MISOKU_SYNC=0",
		GitNotListed:        "this folder is not listed in sync.vaults",
		GitMissing:          "git did not run (%v); install git %s or newer to sync this vault",
		GitUnknownVersion:   "of unknown version",
		GitTooOld:           "git %s is too old; sync needs git %s or newer, update git",
		GitCheckFailed:      "git failed while checking the vault (%v); sync checks again next time",
		GitNotARepoHint:     "this vault is not a git repository (%v); set it up with `misoku sync init <url>`",
		GitNotRepoRoot:      "this vault is inside the git repository at %[1]s; sync needs the vault to be the repository's top folder: open %[1]s instead, or give the vault a repository of its own",
		GitNoRemote:         "the repository has no remote named origin; add one with `git remote add origin <url>`",
		GitInProgress:       "%[1]s is in progress in this vault; finish it (`git %[2]s --continue`) or cancel it (`git %[2]s --abort`)",
		GitOpMerge:          "a merge",
		GitOpRebase:         "a rebase",
		GitOpCherryPick:     "a cherry-pick",
		GitOpRevert:         "a revert",
		GitDetached:         "HEAD is detached (no branch is checked out); check out your branch, for example `git switch main`",
		GitNoIdentity:       "git does not know who you are (%s is not set); run `git config --global user.name \"Your Name\"` and `git config --global user.email you@example.com`",
		GitLocked:           "git's index.lock is there: a git command is running in this vault, or one stopped half-way; wait for it to finish, or if no git is running, delete %s",
		GitLockChanged:      "the lock file changed since it was shown, so it was left alone; check again",
		GitStillRunning:     "still running",
		GitEarlierRunning:   "git (process %d) of an earlier sync is %w; sync tries again once it is done",
		GitNotARepo:         "this vault is not a git repository",
		GitNotInstalled:     "git is not installed",
		GitHostKeyChanged:   "origin's SSH host key changed, which can mean an attack or a reinstalled server: check the key with ssh in a terminal before syncing again",
		GitHostKeyUnknown:   "origin's SSH host key is not known yet: run `misoku sync init <url>` in a terminal once to accept it",
		GitHostKeyRefused:   "origin's SSH host key was not accepted: check that the address is right and accept the key when ssh asks, then run init again",
		GitKeyRefused:       "origin refused your SSH key: add the key to your account (and to ssh-agent), or run `misoku sync init <url>` in a terminal",
		GitKeyRefusedInit:   "origin refused your SSH key: add the key to your account (and to ssh-agent), then run init again",
		GitSignInFailed:     "could not sign in to origin: run `gh auth setup-git` (GitHub over HTTPS) or set up your git credentials, then `misoku sync init <url>` in a terminal if it still fails",
		GitSignInFailedInit: "could not sign in to origin: check the URL, then run `gh auth setup-git` (GitHub over HTTPS) or set up your git credentials, then run init again",
		GitNoReason:         "no reason given",
		GitCouldNotFetch:    "could not fetch from origin",
		GitCouldNotPush:     "could not push to origin",
		GitCouldNotReach:    "could not reach origin",
		GitNeedsLock:        "sync needs the vault's lock",
		GitCountFailed:      "could not count the commits to push: %w",
		GitPushTurnedDown:   "the push was turned down",
		GitSyncStopped:      "sync stopped",
		GitPushedFirst:      "another device pushed first; the next sync joins its changes",
		GitPushRefused:      "origin refused the push (%s); nothing was lost, fix what origin says and sync again: %w",
		GitRewritten:        "origin's %[1]s was rewritten (a force-push), so %[2]s to lose nothing; compare it with `git fetch origin` and `git log origin/%[1]s`, and join it by hand",
		GitUnrelated:        "this vault and origin have unrelated histories; join them once with `misoku sync init <url>`",
		GitNoDefaultBranch:  "origin did not say which branch is its default",
		GitFileBusy:         "a file could not be replaced, perhaps because it is open in another app, so origin's changes were not all applied yet (nothing was lost; close it and sync again): %w",
		GitApplyFailed:      "could not apply origin's changes: %w",
		GitBigClash:         "%[1]s is larger than %[2]s, so it stays out of sync, and origin has a file with the same name: move or rename %[1]s, then sync again: %[3]w",
		GitNoticeClash:      "%s; origin has a file at the same path, so its changes cannot be applied until you move or rename %s: %w",
		GitFileChanged:      "a file changed while syncing, so origin's changes were not applied yet (nothing was lost; the next sync tries again): %w",
		GitNoAnswerAfter:    "no answer after %s",
		GitCancelled:        "cancelled",
		GitCallerTimeout:    "the caller's time limit ran out",
		GitNoGitDir:         "could not find the git folder: %w",
		GitUnsafeFolder:     "%s is your home folder, the root of a disk or a system folder: sync a folder of its own for your notes, such as ~/Notes (misoku sync setup --dir ~/Notes).",
		GitAnotherWindow:    "another Misoku window is syncing this vault; close it, then run `misoku sync init <url>` again",
		GitRemoveOrigin:     "could not remove origin again: %w",
		GitOtherOrigin:      "this repository already has origin %[1]s, not %[2]s; init changed nothing. To sync with %[2]s, run `git remote set-url origin <url>` first, or choose another folder",
		GitThenInitAgain:    "%s; then run `misoku sync init <url>` again",
		GitTracksFiles:      "the git repository at %s tracks files in this folder (%s)",
		GitSubmoduleOf:      "this folder is a submodule of the git repository at %s",
		GitTwoRepos:         "%[1]s, so a repository of its own would put your notes in two repositories; %[2]s changed nothing. Move the notes to a folder outside %[3]s, then run %[2]s again",
		GitInsideRepo:       "this folder is inside the git repository at %[1]s, and syncing it would commit that whole repository; %[2]s changed nothing. Give the vault a repository of its own: move it outside %[1]s, or run `git init` in this folder (the outer repository then sees it as a nested one), then run %[2]s again; `misoku sync setup` offers to run `git init` for you",
		GitNotAFolder:       "%s is not a folder",
		GitNoParent:         "%s does not exist, nor does its folder %s; create it first, or choose another folder",
		GitInitDidNotRun:    "git did not run (%w); install git %s or newer",
		GitInitTooOld:       "git %s is too old; sync needs git %s or newer, update git",
		GitReportRepo:       "Created a git repository (branch %s).",
		GitReportOrigin:     "Added origin %s.",
		GitReportAttrs:      "Wrote .gitattributes (* -text): files stay byte for byte, with no line-ending conversion.",
		GitReportKeptAttrs:  "Kept your .gitattributes as it is; add the line `* -text` to it so line endings never change between systems.",
		GitReportLeftOut:    "Left out %s (%d MB): files that big do not sync.",

		GitReportCommitted + one:   "Committed %d file.",
		GitReportCommitted + other: "Committed %d files.",
		GitReportBrought:           "Brought origin's notes.",
		GitReportConflicts:         "Kept both versions of %d note(s) changed on both sides; the other version is in:",
		GitReportResolve:           "Resolve each by deleting the copy (or copying what you want into the note).",
		GitReportPushed:            "Pushed to origin (branch %s).",
		GitCreatedFolder:           "Created the folder %s.",
		GitAddedToVaults:           "Added %s to sync.vaults in %s.",
		GitNoticeNested:            "%s is a separate git repository, so its files do not sync: move it out of the vault, or ignore it",
		GitNoticeSubmodule:         "%s is a git submodule: only its commit id syncs, not its files",
		GitNoticeLink:              "%s links to %s, outside the vault: only the link syncs, not what it points to",
		GitNoticeWinLeftOut:        "%s has a name Windows cannot use (%s), so it stays out of sync until you rename it to a valid name; a note renamed to it keeps syncing under its old name",
		GitNoticeWinName:           "%s has a name Windows cannot use (%s): rename it, or a Windows computer cannot sync this vault",
		GitNoticeCaseClash:         "%s and another note or folder have names that differ only in letter case, so this computer holds only one of them: rename the file shown here and both sync again on every device",
		GitNoticeCaseDup:           "%s has the name of another file, ignoring letter case, which macOS and Windows cannot hold together, so it stays out of sync until you rename it",
		GitNoticeHeld:              "%s was deleted or renamed, but it keeps syncing under this name until the files left out for their names are renamed",
		GitNameControl:             "a control character",
		GitNameDotSpace:            "it ends with a dot or a space",
		GitNameReserved:            "%s is reserved on Windows",

		CoreNotFound:          "page not found",
		CoreNoTrash:           "no trash on this system",
		CoreBadWorkspace:      "invalid workspace path: %w",
		CoreMkWorkspace:       "failed to create workspace directory: %w",
		CoreWorkspaceAccess:   "failed to access workspace directory: %w",
		CoreNotADir:           "workspace path %q is not a directory",
		CoreNameEmpty:         "file name is empty",
		CoreNameChars:         `file name cannot contain / \ : * ? " < > |`,
		CoreNameReserved:      "%s is a reserved name on Windows",
		CoreOutside:           "location must be inside the workspace",
		CoreHidden:            "hidden files and folders are not allowed",
		CoreExists:            "%s already exists",
		CoreMkFolder:          "failed to create folder: %w",
		CoreSaveFailed:        "failed to save page: %w",
		CoreLinksFailed:       "failed to update links in %s: %w",
		CoreIsFolder:          "%s is a folder",
		CoreMkAssets:          "failed to create assets folder: %w",
		CoreCopyFailed:        "failed to copy %s: %w",
		CoreMoveRoot:          "the workspace root cannot be moved",
		CoreMoveInto:          "cannot move %q into itself",
		CoreAlreadyThere:      "%q is already in that folder",
		CoreExistsThere:       "%q already exists in the destination",
		CoreMoveFolder:        "failed to move folder: %w",
		CoreMovePage:          "failed to move page: %w",
		CoreMoveSubNothing:    "failed to move sub-pages, nothing moved: %w",
		CoreMoveSubHalf:       "moved the page to %s but not its sub-pages folder: %w",
		CoreMoveReadBack:      "moved the page but could not read it back: %w",
		CoreBackslash:         `"\" is not allowed in a name`,
		CoreTrashMissing:      "cannot find the Trash: %w",
		CoreTrashFailed:       "could not move to the Trash: %w",
		CoreTrashFull:         "could not move to the Trash: too many items named %s",
		CoreCreateFailed:      "failed to create page: %w",
		CoreRelativeFolder:    "the path must be relative to the folder",
		CoreRelativeWorkspace: "the path must be relative to the workspace",
		CoreEmptySegment:      "the path has an empty name",
		CoreDots:              `"." and ".." are not allowed`,
		CorePageExists:        "a page named %q already exists here",
		CoreFolderExists:      "a folder named %q already exists here",
		CoreRenameFailed:      "failed to rename page: %w",
		CoreRenameSubBack:     "failed to rename sub-pages and could not put the note back: %v: %w",
		CoreRenameSubNothing:  "failed to rename sub-pages, nothing moved: %w",
		CoreRenameFolder:      "failed to rename folder: %w",
		CoreDeleteFolder:      "failed to delete folder: %w",
		CoreDeletePage:        "failed to delete page: %w",
		CoreDeleteSubHalf:     "deleted the page but not its sub-pages folder: %w",

		GraphNoNotes:                  "No notes yet",
		GraphTooSmall:                 "Graph needs a bigger pane",
		GraphAllNotes:                 "All notes",
		GraphLinks + one:              "%d link",
		GraphLinks + other:            "%d links",
		GraphBacklinks + one:          "%d backlink",
		GraphBacklinks + other:        "%d backlinks",
		GraphNotes + one:              "%d note",
		GraphNotes + other:            "%d notes",
		GraphSteps + one:              "%d step",
		GraphSteps + other:            "%d steps",
		GraphOrphans + one:            "%d orphan",
		GraphOrphans + other:          "%d orphans",
		GraphUnresolved + one:         "%d unresolved",
		GraphUnresolved + other:       "%d unresolved",
		GraphMatches + one:            "%d match",
		GraphMatches + other:          "%d matches",
		GraphOrphansHidden + one:      "%d orphan hidden",
		GraphOrphansHidden + other:    "%d orphans hidden",
		GraphUnresolvedHidden + one:   "%d unresolved hidden",
		GraphUnresolvedHidden + other: "%d unresolved hidden",
		GraphFilterOn:                 " · filter ",
		GraphCentre:                   "centre",
		GraphBack:                     "back",
		GraphReset:                    "reset",
		GraphFilter:                   "filter",
		GraphOrphansBtn:               "orphans",
		GraphUnresBtn:                 "unresolved",
		GraphTailFull:                 "  hjkl move · enter open · click: preview, again: open · scroll: zoom · drag: move",
		GraphTailDrag:                 "  hjkl move · enter open · drag: move",
		GraphTailOpen:                 "  hjkl move · enter open",
		GraphTailKeys:                 "  hjkl · enter",
		GraphMapTailFull:              "  hjkl move · n/N links · enter open · click: preview, again: open · scroll: zoom · drag: move",
		GraphMapTailDrag:              "  hjkl move · n/N links · enter open · drag: move",
		GraphMapTailOpen:              "  hjkl move · n/N links · enter open",
		GraphPromptJump:               "   enter jump · esc clear",
	})
	register(ES, map[Key]string{
		CliUsage: `Uso:
  misoku [carpeta] [opciones]              abre tus notas: la carpeta indicada, si no $MISOKU_WORKSPACE,
                                           si no notes_dir de la configuración, si no la carpeta actual
  misoku export <nota.md> [-o salida.pdf]  exporta una nota a PDF
  misoku paste-image [--dir <carpeta>]     guarda la imagen del portapapeles en <carpeta>/assets
                                           y muestra la ruta para enlazarla (--link muestra el enlace)
  misoku sync setup [--dir <carpeta>]      sincroniza una carpeta con un repositorio privado de GitHub (recomendado)
  misoku sync init <url> [--dir <carpeta>] prepara el sync con git de una carpeta con cualquier URL de git (SSH, otros servicios)
  misoku tutor [--route B|I|A|E] [--reset]  aprende Misoku y Markdown por rutas (--reset empieza de nuevo)

Opciones:
`,
		CliFlagDir:           "carpeta que abrir, como misoku <carpeta> (por defecto $MISOKU_WORKSPACE, luego notes_dir de la configuración, luego la carpeta actual)",
		CliFlagVersion:       "muestra la versión de Misoku y termina",
		CliFlagTheme:         "tema de colores para esta sesión (mira --list-themes)",
		CliFlagThemesDir:     "carpeta o archivo .json con temas propios (reemplaza themes_dir de la configuración)",
		CliFlagListThemes:    "lista los temas de colores disponibles y termina",
		CliIgnoringConfig:    "Se ignora la configuración ilegible %s: %v",
		CliIgnoring:          "Se ignora %v",
		CliThemesPathMissing: "No se encontró la ruta de temas propios %s; solo se usan los temas incluidos",
		CliUnknownTheme:      "Tema desconocido %q (ejecuta misoku --list-themes)",
		CliOpenFailed:        "No se pudo abrir la carpeta de notas %q: %v",
		CliLogFailed:         "No se pudo abrir el archivo de registro: %v",
		CliRunFailed:         "Error al ejecutar Misoku: %v",
		CliNotesMissing:      "Falta tu carpeta de notas %s; abre otra con misoku <carpeta>",

		CliExportUsage:   "Uso: misoku export <nota.md> [-o salida.pdf]",
		CliExportFlagOut: "archivo o carpeta de salida (por defecto el nombre de la nota con .pdf, a su lado)",
		CliReadFailed:    "No se pudo leer %s: %v",
		CliExportFailed:  "No se pudo exportar: %v",

		CliPasteUsage:    "Uso: misoku paste-image [--dir <carpeta>] [--link]",
		CliPasteFlagDir:  "carpeta de la nota a la que pertenece la imagen",
		CliPasteFlagLink: "muestra el enlace de Markdown en lugar de la ruta",
		CliPasteNotDir:   "%s no es una carpeta",
		CliPasteNeeds:    "necesita %s",
		CliPasteNoImage:  "el portapapeles no tiene ninguna imagen",
		CliPasteMkdir:    "no se pudo crear %s: %w",
		CliPasteNoName:   "no se encontró un nombre libre para %s",
		CliPasteWrite:    "no se pudo escribir %s: %w",

		CliTutorUsage:       "Uso: misoku tutor [--route B|I|A|E] [--reset]",
		CliTutorFlagReset:   "empieza el tutorial de nuevo desde las lecciones incluidas",
		CliTutorFlagRoute:   "ruta que abrir (B, I, A o E)",
		CliTutorUnexpected:  "argumento inesperado %q",
		CliTutorRouteNeeds:  "--route necesita B, I, A o E",
		CliTutorRouteBad:    "ruta desconocida %q (se espera B, I, A o E)",
		CliTutorGitFailed:   "no se pudo preparar el historial de git del tutorial: %v",
		CliTutorBadProgress: "se ignora el archivo de avance: %v",

		CliSyncInitUsage:   "Uso: misoku sync init <url> [--dir <carpeta>]",
		CliSyncSetupUsage:  "Uso: misoku sync setup [--dir <carpeta>]",
		CliSyncSetupLine:   "     misoku sync setup [--dir <carpeta>]   (recomendado: conecta un repositorio privado de GitHub por ti)",
		CliSyncFlagDir:     "carpeta del vault (por defecto $MISOKU_WORKSPACE, luego la carpeta actual)",
		CliInitBadConfig:   "no se pudo leer el archivo de configuración %s (%v); corrígelo y vuelve a ejecutar init",
		CliInitSettingUp:   "Preparando el sync de %s con %s…",
		CliInitUnencrypted: "Tus notas se guardan sin cifrar en ese repositorio: mantenlo privado.",
		CliInitNothingLost: "%v\nNo se perdió nada; corrige eso y vuelve a ejecutar `misoku sync init <url>`.",
		CliInitNotListed:   "el vault está listo, pero no se pudo añadir a la configuración (%v); añádelo a mano en sync.vaults:\n  {\"sync\": {\"vaults\": {%s: {}}}}",
		CliInitAlready:     "Ya estaba listo: esta carpeta se sincroniza con %s (rama %s); no cambió nada.",
		CliInitDone:        "Listo: Misoku sincroniza esta carpeta al abrirla y con :sync.",
		CliSetupSettingUp:  "Preparando el sync de %s con GitHub.",
		CliSetupStart:      "¿Empezar?",
		CliSetupName:       "Tu nombre",
		CliSetupEmail:      "Tu email",
		CliSetupSignInNow:  "¿Iniciar sesión en GitHub ahora?",
		CliSetupRepoName:   "Nombre del repositorio",
		CliSetupPickRepo:   "Número, nombre o parte de uno (Enter cancela): ",
		CliSetupMatching:   "Repositorios que coinciden con %q:",
		CliSetupPrivate:    "privado",
		CliSetupNotAllowed: "%s: no permitido, las notas se guardan sin cifrar",
		CliYesNo:           " [s/N] ",
		CliYesDefault:      " [S/n] ",
		CliYesWords:        "s si sí y yes",

		WelcomeTitle:         "Te damos la bienvenida a Misoku.",
		WelcomeMissing:       "No encuentro tus notas en %s.",
		WelcomeNotesQuestion: "¿Dónde están tus notas?",
		WelcomeStart:         "Empezar mis notas",
		WelcomeBring:         "Traer mis notas de otro equipo",
		WelcomeUseThis:       "Usar esta carpeta: %s",
		WelcomeNumber:        "Número (Enter cancela): ",
		WelcomeType:          "! Escribe %s.",
		WelcomeOneOrTwo:      "1 o 2",
		WelcomeOneTwoThree:   "1, 2 o 3",
		WelcomeStopped:       "No se cambió nada. Vuelve a ejecutar misoku cuando quieras.",
		WelcomeFolderAsk:     "Carpeta para tus notas [%s]: ",
		WelcomeFolder:        "Carpeta para tus notas:",
		WelcomeFolderShown:   "Carpeta para tus notas: %s",
		WelcomeFullPath:      "Escribe una ruta completa, como ~/Notas.",
		WelcomeUnsafe:        "%s es tu carpeta personal, la raíz de un disco o una carpeta del sistema: elige una carpeta propia para tus notas, como ~/Notas.",
		WelcomeIsFile:        "%s es un archivo, no una carpeta: elige otro nombre",
		WelcomeNoParent:      "la carpeta %s no existe: créala primero o elige otra carpeta",
		WelcomeCreateFailed:  "no se pudo crear %s (%w)",
		WelcomeNotRemembered: "No se pudo recordar %s como tu carpeta de notas (%v); misoku volverá a preguntar la próxima vez.",
		WelcomeShowFailed:    "No se pudo mostrar la bienvenida (%v).",
		WelcomeHintChoose:    "↑↓ o un número · Enter elegir · Esc o q salir",
		WelcomeHintCancel:    "Esc cancelar",
		WelcomeHintCreate:    "Enter crear y abrir · Esc salir",
		WelcomeHintBring:     "Enter traer las notas aquí · Esc salir",

		SetupBadConfig:        "No se pudo leer el archivo de configuración %s (%v); corrígelo y vuelve a ejecutar el setup.",
		SetupBringTitle:       "Trae tus notas",
		SetupBringLine:        "Tus notas llegan del repositorio privado de GitHub con el que se sincroniza tu otro equipo, y desde entonces se mantienen sincronizadas.",
		SetupSyncTitle:        "Sincronizar este vault",
		SetupSyncLine:         "Tus notas se guardarán en un repositorio privado de GitHub y llegarán solas a tus otros equipos.",
		SetupPrivateLine:      "En el repositorio tus notas no están cifradas, así que se queda privado.",
		SetupGHAccess:         "Necesitas una cuenta de GitHub. Misoku inicia sesión con gh, la herramienta de línea de comandos de GitHub, que guarda el acceso: Misoku nunca lo ve ni lo guarda.",
		SetupCreateChoice:     "Crear un repositorio privado nuevo",
		SetupFreshChoice:      "No tengo notas en otro equipo → Empezar mis notas",
		SetupNoParent:         "%s no existe, ni tampoco su carpeta %s; créala primero o elige otra carpeta.",
		SetupNotFolder:        "%s no es una carpeta.",
		SetupAlreadySynced:    "Este vault ya está listo: se sincroniza con %s; no cambió nada.",
		SetupPausedThough:     "Pero el sync está en pausa: %s.",
		SetupStatusHint:       "`:sync status` en Misoku muestra cómo va.",
		SetupTitle:            "Setup del sync",
		SetupAlreadyRepo:      "Esta carpeta ya está preparada con el repositorio de git %s, aunque Misoku todavía no la sincroniza; no cambió nada.",
		SetupAlreadyRepoHint:  "Para sincronizarla con ese repositorio, ejecuta `misoku sync init %s`.",
		SetupOwnRepoTitle:     "Un repositorio para tus notas",
		SetupOwnRepoLine:      "Esta carpeta está dentro del repositorio de git de %s, que no la sigue: sincronizarla así haría commit de todo ese repositorio. Un repositorio aparte deja ese como está.",
		SetupOwnRepoQuestion:  "¿Crear aquí un repositorio aparte para tus notas?",
		SetupLookFailed:       "El setup no pudo revisar %s (%v). El setup no cambió nada.",
		SetupRemoveFailed:     "El setup creó la carpeta vacía %s para revisarla y no pudo quitarla de nuevo (%v); bórrala y vuelve a ejecutar el setup.",
		SetupChangedNothing:   "%s. El setup no cambió nada.",
		SetupCancelled:        "Setup cancelado",
		SetupEnded:            "El setup ya terminó; vuelve a ejecutarlo.",
		SetupInitFailed:       "git no pudo crear un repositorio en esta carpeta (%v).",
		SetupMadeOwnRepo:      "Se creó un repositorio de git propio en esta carpeta",
		SetupMadeOwnRepoNote:  "✓ se creó un repositorio de git propio para tus notas",
		SetupMadeFolder:       "Se creó la carpeta %s",
		SetupMadeRepo:         "Se creó un repositorio de git en %s",
		SetupLeftNotEmpty:     "; se dejó en su sitio porque no está vacía",
		SetupNotRemoved:       "; no se pudo quitar de nuevo (%v)",
		SetupFolderRemoved:    "Se quitó la carpeta %s que había creado",
		SetupLeftUnknown:      "; se dejó en su sitio porque git no pudo decir si tiene commits (%v)",
		SetupLeftCommits:      "; ya tiene commits, así que se dejó en su sitio",
		SetupLeftNotOurs:      "; se dejó en su sitio porque %s ya no es la carpeta que creó",
		SetupRepoNotRemoved:   "; no se pudo quitar de nuevo (%v): borra %s a mano",
		SetupRepoRemoved:      "Se quitó el repositorio de git que había creado",
		SetupAppleGit:         "Apple está instalando git; pulsa Enter cuando termine.",
		SetupGitDidNotRun:     "git no se ejecutó (%v). Instala git %s o más nuevo y vuelve a ejecutar el setup.",
		SetupGitTooOld:        "git %s es demasiado antiguo: el sync necesita git %s o más nuevo. Actualiza git y vuelve a ejecutar el setup.",
		SetupNoGH:             "gh, la herramienta de línea de comandos de GitHub, no está instalada: el setup la usa para iniciar sesión en GitHub.",
		SetupIdentityLine:     "git todavía no conoce tu nombre ni tu email: los escribe en cada cambio que registra (el historial de tus notas).",
		SetupIdentityGitHub:   "Misoku los completó con tu cuenta de GitHub (el email es la dirección privada que GitHub te da); cámbialos si quieres.",
		SetupIdentityTitle:    "Tu nombre para git",
		SetupIdentitySave:     "¿Guardarlos en los ajustes globales de git (git config --global), para todos los repositorios de este equipo?",
		SetupTypeName:         "Escribe tu nombre (cualquiera: solo se muestra en el historial).",
		SetupTypeEmail:        "Escribe un email como tu@ejemplo.com.",
		SetupNameNotSaved:     "git no pudo guardar user.name (%v).",
		SetupEmailNotSaved:    "git no pudo guardar user.email (%v).",
		SetupNameSaved:        "Se guardó tu nombre de git",
		SetupIdentitySaved:    "Se guardaron tu nombre y tu email de git",
		SetupKnowsYou:         "✓ git te conoce: %s <%s>",
		SetupSignInLine:       "Inicia sesión en GitHub: gh muestra un código de un solo uso y abre tu navegador.",
		SetupSignInKeychain:   "Inicia sesión ahí y escribe el código. gh guarda el acceso en el llavero de tu sistema; Misoku nunca lo ve.",
		SetupNoBrowser:        "Este equipo no tiene un navegador que gh pueda abrir: cuando gh muestre la dirección https://github.com/login/device, ábrela en cualquier dispositivo (sirve tu teléfono), inicia sesión y escribe el código.",
		SetupSignInTitle:      "Iniciar sesión en GitHub",
		SetupSignedIn:         "✓ sesión iniciada en GitHub",
		SetupInstallMac:       "Instálalo con `brew install gh` (mira https://cli.github.com) y vuelve a ejecutar el setup.",
		SetupInstallWindows:   "Instálalo con `winget install --id GitHub.cli` (mira https://cli.github.com) y vuelve a ejecutar el setup.",
		SetupInstallLinux:     "Instálalo con tu gestor de paquetes, por ejemplo `sudo apt install gh` (mira https://cli.github.com), y vuelve a ejecutar el setup.",
		SetupNotSignedIn:      "No iniciaste sesión en GitHub (el inicio de sesión no terminó). Vuelve a ejecutar el setup para intentarlo otra vez.",
		SetupListFailed:       "gh no pudo listar tus repositorios (%v).",
		SetupListUnreadable:   "gh listó tus repositorios de una forma que Misoku no entiende (%v).",
		SetupWhichBring:       "¿Qué repositorio tiene tus notas?",
		SetupPickBring:        "Elige el repositorio con el que se sincroniza tu otro equipo.",
		SetupWhich:            "¿Qué repositorio guarda este vault?",
		SetupPick:             "Elige el repositorio con el que ya se sincroniza tu otro equipo, o crea uno privado nuevo.",
		SetupNotYours:         "%s no es uno de tus repositorios.",
		SetupNoNotes:          "No hay notas que traer",
		SetupStartTitle:       "Empezar mis notas",
		SetupNewTitle:         "Nombre del repositorio privado nuevo",
		SetupNewLine:          "Misoku lo crea privado en GitHub y sube tus notas a él.",
		SetupChooseListed:     "Elige uno de los repositorios de la lista.",
		SetupPublic:           "%s es público: cualquiera podría leer tus notas, que se guardan sin cifrar. Elige un repositorio privado o crea uno nuevo (o hazlo privado antes en GitHub).",
		SetupNameRule:         "Usa solo letras, números, '-', '_' y '.' (la regla de GitHub).",
		SetupNameYours:        "Ya tienes un repositorio llamado %s: elígelo de la lista (vuelve a ejecutar el setup) o elige otro nombre.",
		SetupNameTaken:        "Ya existe un repositorio llamado %s en tu cuenta de GitHub; elige otro nombre.",
		SetupCreateFailed:     "GitHub no creó el repositorio (%v).",
		SetupCreatedOnGitHub:  "se creó el repositorio privado %s en GitHub",
		SetupCreatedNowhere:   "GitHub creó el repositorio %s, pero gh no dijo dónde está. Vuelve a ejecutar el setup y elígelo de la lista.",
		SetupGitAccessFailed:  "gh no pudo darle a git el acceso a GitHub (%v)",
		SetupGitAccess:        "Se preparó con gh el acceso de git a GitHub",
		SetupCreated:          "✓ Se creó el repositorio privado %s.",
		SetupFixAndPick:       "El repositorio %s está en GitHub: corrige eso, vuelve a ejecutar el setup y elígelo.",
		SetupNothingLost:      "%s; no se perdió nada.",
		SetupNotListed:        "El vault está listo, pero no se pudo añadir a la configuración (%v); añádelo a mano en sync.vaults: {\"sync\": {\"vaults\": {%s: {}}}}",
		SetupWhileOpen:        "mientras está abierto",
		SetupEvery:            "cada %s mientras está abierto",
		SetupDone:             "Listo: Misoku sincroniza este vault con %s %s, al abrirlo y al salir.",
		SetupOtherComputer:    "En tu otro equipo: instala Misoku, ejecuta misoku y elige \"%s\"; luego elige %s.",
		SetupDoneTitle:        "El sync está listo",
		SetupDidNotFinish:     "el sync no terminó",
		SetupNothingChanged:   "No se cambió nada.",
		SetupStopped:          "El setup se detuvo. %s",
		SetupStopNothing:      "%s; no se cambió nada.",
		SetupStopChanged:      "%s. %s",
		SetupChanges:          "%s. No cambió nada más.",
		SetupStoppedTitle:     "El setup del sync se detuvo",
		SetupNoAnswer:         "gh %s: sin respuesta tras %s",
		SetupSignInUnfinished: "el inicio de sesión no terminó",
		SetupReadFailed:       "Misoku no pudo leer %s para revisarlo antes de usarlo (%s).",
		SetupNothingFetched:   "No se trajo nada a esta carpeta ni se subió nada. Revisa tu conexión y tu acceso a él, y vuelve a ejecutar el setup.",
		SetupCodeProject:      "%s parece un proyecto de código (%s): Misoku no mezclará tus notas con él. Elige otro repositorio o crea uno privado nuevo.",
		SetupAndMore:          " y %d más",
		SetupUnrelated:        "%s no comparte el historial de esta carpeta. Contiene: %s%s.",
		SetupNotNotes:         "%d de %d archivos no son notas",
		SetupMixLine:          "Usarlo junta los dos: sus archivos llegan a esta carpeta y tus notas se suben a él.",
		SetupMixCancel:        "Si no es el sitio de tus notas, cancela y crea un repositorio privado nuevo.",
		SetupMixTitle:         "Este repositorio ya tiene otros archivos",
		SetupMixQuestion:      "¿Mezclar las notas de esta carpeta con %s?",
		SetupOtherFiles:       "Puede que no sea un repositorio de notas: %s.",

		GitOff:              "el sync está apagado por MISOKU_SYNC=0",
		GitNotListed:        "esta carpeta no está en sync.vaults",
		GitMissing:          "git no se ejecutó (%v); instala git %s o más nuevo para sincronizar este vault",
		GitUnknownVersion:   "de versión desconocida",
		GitTooOld:           "git %s es demasiado antiguo; el sync necesita git %s o más nuevo, actualiza git",
		GitCheckFailed:      "git falló al revisar el vault (%v); el sync lo revisa de nuevo la próxima vez",
		GitNotARepoHint:     "este vault no es un repositorio de git (%v); prepáralo con `misoku sync init <url>`",
		GitNotRepoRoot:      "este vault está dentro del repositorio de git de %[1]s; el sync necesita que el vault sea la carpeta principal del repositorio: abre %[1]s en su lugar, o dale al vault un repositorio propio",
		GitNoRemote:         "el repositorio no tiene un remoto llamado origin; añade uno con `git remote add origin <url>`",
		GitInProgress:       "hay %[1]s a medias en este vault; termínalo (`git %[2]s --continue`) o cancélalo (`git %[2]s --abort`)",
		GitOpMerge:          "un merge",
		GitOpRebase:         "un rebase",
		GitOpCherryPick:     "un cherry-pick",
		GitOpRevert:         "un revert",
		GitDetached:         "HEAD está separado (no hay ninguna rama activa); cambia a tu rama, por ejemplo con `git switch main`",
		GitNoIdentity:       "git no sabe quién eres (%s no está definido); ejecuta `git config --global user.name \"Tu Nombre\"` y `git config --global user.email tu@ejemplo.com`",
		GitLocked:           "está el index.lock de git: un comando de git se está ejecutando en este vault, o uno se detuvo a medias; espera a que termine o, si no hay ningún git en marcha, borra %s",
		GitLockChanged:      "el archivo de bloqueo cambió desde que se mostró, así que no se tocó; vuelve a revisarlo",
		GitStillRunning:     "sigue en marcha",
		GitEarlierRunning:   "git (proceso %d) de un sync anterior %w; el sync lo intenta de nuevo cuando termine",
		GitNotARepo:         "este vault no es un repositorio de git",
		GitNotInstalled:     "git no está instalado",
		GitHostKeyChanged:   "la clave SSH del servidor de origin cambió, lo que puede ser un ataque o un servidor reinstalado: revisa la clave con ssh en una terminal antes de volver a sincronizar",
		GitHostKeyUnknown:   "la clave SSH del servidor de origin aún no es conocida: ejecuta una vez `misoku sync init <url>` en una terminal para aceptarla",
		GitHostKeyRefused:   "no se aceptó la clave SSH del servidor de origin: revisa que la dirección sea correcta y acepta la clave cuando ssh pregunte; luego vuelve a ejecutar init",
		GitKeyRefused:       "origin rechazó tu clave SSH: añade la clave a tu cuenta (y a ssh-agent), o ejecuta `misoku sync init <url>` en una terminal",
		GitKeyRefusedInit:   "origin rechazó tu clave SSH: añade la clave a tu cuenta (y a ssh-agent) y vuelve a ejecutar init",
		GitSignInFailed:     "no se pudo iniciar sesión en origin: ejecuta `gh auth setup-git` (GitHub por HTTPS) o configura tus credenciales de git; luego `misoku sync init <url>` en una terminal si sigue fallando",
		GitSignInFailedInit: "no se pudo iniciar sesión en origin: revisa la URL, ejecuta `gh auth setup-git` (GitHub por HTTPS) o configura tus credenciales de git, y vuelve a ejecutar init",
		GitNoReason:         "sin motivo indicado",
		GitCouldNotFetch:    "no se pudo traer de origin",
		GitCouldNotPush:     "no se pudo subir a origin",
		GitCouldNotReach:    "no se pudo conectar con origin",
		GitNeedsLock:        "el sync necesita el bloqueo del vault",
		GitCountFailed:      "no se pudieron contar los commits por subir: %w",
		GitPushTurnedDown:   "se rechazó la subida",
		GitSyncStopped:      "el sync se detuvo",
		GitPushedFirst:      "otro dispositivo subió primero; el próximo sync junta sus cambios",
		GitPushRefused:      "origin rechazó la subida (%s); no se perdió nada, corrige lo que dice origin y vuelve a sincronizar: %w",
		GitRewritten:        "la rama %[1]s de origin se reescribió (un force-push), así que %[2]s para no perder nada; compárala con `git fetch origin` y `git log origin/%[1]s`, y júntala a mano",
		GitUnrelated:        "este vault y origin tienen historiales sin relación; júntalos una vez con `misoku sync init <url>`",
		GitNoDefaultBranch:  "origin no dijo cuál es su rama por defecto",
		GitFileBusy:         "no se pudo reemplazar un archivo, quizás porque está abierto en otra app, así que aún no se aplicaron todos los cambios de origin (no se perdió nada; ciérralo y vuelve a sincronizar): %w",
		GitApplyFailed:      "no se pudieron aplicar los cambios de origin: %w",
		GitBigClash:         "%[1]s pesa más de %[2]s, así que no se sincroniza, y origin tiene un archivo con el mismo nombre: mueve o renombra %[1]s y vuelve a sincronizar: %[3]w",
		GitNoticeClash:      "%s; origin tiene un archivo en la misma ruta, así que sus cambios no se pueden aplicar hasta que muevas o renombres %s: %w",
		GitFileChanged:      "un archivo cambió durante el sync, así que aún no se aplicaron los cambios de origin (no se perdió nada; el próximo sync lo intenta de nuevo): %w",
		GitNoAnswerAfter:    "sin respuesta tras %s",
		GitCancelled:        "cancelado",
		GitCallerTimeout:    "se agotó el tiempo de quien lo llamó",
		GitNoGitDir:         "no se encontró la carpeta de git: %w",
		GitUnsafeFolder:     "%s es tu carpeta personal, la raíz de un disco o una carpeta del sistema: sincroniza una carpeta propia para tus notas, como ~/Notas (misoku sync setup --dir ~/Notas).",
		GitAnotherWindow:    "otra ventana de Misoku está sincronizando este vault; ciérrala y vuelve a ejecutar `misoku sync init <url>`",
		GitRemoveOrigin:     "no se pudo quitar origin de nuevo: %w",
		GitOtherOrigin:      "este repositorio ya tiene el origin %[1]s, no %[2]s; init no cambió nada. Para sincronizar con %[2]s, ejecuta antes `git remote set-url origin <url>`, o elige otra carpeta",
		GitThenInitAgain:    "%s; luego vuelve a ejecutar `misoku sync init <url>`",
		GitTracksFiles:      "el repositorio de git de %s sigue archivos de esta carpeta (%s)",
		GitSubmoduleOf:      "esta carpeta es un submódulo del repositorio de git de %s",
		GitTwoRepos:         "%[1]s, así que un repositorio propio pondría tus notas en dos repositorios; %[2]s no cambió nada. Mueve las notas a una carpeta fuera de %[3]s y vuelve a ejecutar %[2]s",
		GitInsideRepo:       "esta carpeta está dentro del repositorio de git de %[1]s, y sincronizarla haría commit de todo ese repositorio; %[2]s no cambió nada. Dale al vault un repositorio propio: muévelo fuera de %[1]s, o ejecuta `git init` en esta carpeta (el repositorio de fuera la verá como uno anidado), y vuelve a ejecutar %[2]s; `misoku sync setup` ofrece ejecutar `git init` por ti",
		GitNotAFolder:       "%s no es una carpeta",
		GitNoParent:         "%s no existe, ni tampoco su carpeta %s; créala primero o elige otra carpeta",
		GitInitDidNotRun:    "git no se ejecutó (%w); instala git %s o más nuevo",
		GitInitTooOld:       "git %s es demasiado antiguo; el sync necesita git %s o más nuevo, actualiza git",
		GitReportRepo:       "Se creó un repositorio de git (rama %s).",
		GitReportOrigin:     "Se añadió el origin %s.",
		GitReportAttrs:      "Se escribió .gitattributes (* -text): los archivos quedan byte a byte, sin convertir los finales de línea.",
		GitReportKeptAttrs:  "Se dejó tu .gitattributes como está; añádele la línea `* -text` para que los finales de línea nunca cambien entre sistemas.",
		GitReportLeftOut:    "Se dejó fuera %s (%d MB): los archivos tan grandes no se sincronizan.",

		GitReportCommitted + one:   "Commit de %d archivo.",
		GitReportCommitted + other: "Commit de %d archivos.",
		GitReportBrought:           "Se trajeron las notas de origin.",
		GitReportConflicts:         "Se guardaron las dos versiones de %d nota(s) cambiadas en los dos lados; la otra versión está en:",
		GitReportResolve:           "Resuelve cada una borrando la copia (o copiando a la nota lo que quieras).",
		GitReportPushed:            "Se subió a origin (rama %s).",
		GitCreatedFolder:           "Se creó la carpeta %s.",
		GitAddedToVaults:           "Se añadió %s a sync.vaults en %s.",
		GitNoticeNested:            "%s es un repositorio de git aparte, así que sus archivos no se sincronizan: sácalo del vault o ignóralo",
		GitNoticeSubmodule:         "%s es un submódulo de git: solo se sincroniza el id de su commit, no sus archivos",
		GitNoticeLink:              "%s enlaza a %s, fuera del vault: solo se sincroniza el enlace, no a lo que apunta",
		GitNoticeWinLeftOut:        "%s tiene un nombre que Windows no puede usar (%s), así que no se sincroniza hasta que le pongas un nombre válido; una nota renombrada así sigue sincronizándose con su nombre anterior",
		GitNoticeWinName:           "%s tiene un nombre que Windows no puede usar (%s): renómbralo, o un equipo con Windows no podrá sincronizar este vault",
		GitNoticeCaseClash:         "%s y otra nota o carpeta tienen nombres que solo se diferencian en mayúsculas y minúsculas, así que este equipo guarda solo uno: renombra el archivo que se muestra aquí y los dos se vuelven a sincronizar en todos los dispositivos",
		GitNoticeCaseDup:           "%s tiene el nombre de otro archivo, sin contar mayúsculas y minúsculas, y macOS y Windows no pueden guardar los dos, así que no se sincroniza hasta que lo renombres",
		GitNoticeHeld:              "%s se borró o se renombró, pero sigue sincronizándose con este nombre hasta que se renombren los archivos que quedaron fuera por su nombre",
		GitNameControl:             "un carácter de control",
		GitNameDotSpace:            "termina en un punto o un espacio",
		GitNameReserved:            "%s está reservado en Windows",

		CoreNotFound:          "no se encontró la página",
		CoreNoTrash:           "no hay papelera en este sistema",
		CoreBadWorkspace:      "ruta de la carpeta de notas no válida: %w",
		CoreMkWorkspace:       "no se pudo crear la carpeta de notas: %w",
		CoreWorkspaceAccess:   "no se pudo acceder a la carpeta de notas: %w",
		CoreNotADir:           "la ruta de la carpeta de notas %q no es una carpeta",
		CoreNameEmpty:         "el nombre del archivo está vacío",
		CoreNameChars:         `el nombre del archivo no puede contener / \ : * ? " < > |`,
		CoreNameReserved:      "%s es un nombre reservado en Windows",
		CoreOutside:           "la ubicación debe estar dentro de la carpeta de notas",
		CoreHidden:            "no se permiten archivos ni carpetas ocultos",
		CoreExists:            "%s ya existe",
		CoreMkFolder:          "no se pudo crear la carpeta: %w",
		CoreSaveFailed:        "no se pudo guardar la página: %w",
		CoreLinksFailed:       "no se pudieron actualizar los enlaces de %s: %w",
		CoreIsFolder:          "%s es una carpeta",
		CoreMkAssets:          "no se pudo crear la carpeta de adjuntos: %w",
		CoreCopyFailed:        "no se pudo copiar %s: %w",
		CoreMoveRoot:          "la carpeta de notas no se puede mover",
		CoreMoveInto:          "no se puede mover %q dentro de sí mismo",
		CoreAlreadyThere:      "%q ya está en esa carpeta",
		CoreExistsThere:       "%q ya existe en el destino",
		CoreMoveFolder:        "no se pudo mover la carpeta: %w",
		CoreMovePage:          "no se pudo mover la página: %w",
		CoreMoveSubNothing:    "no se pudieron mover las subpáginas, no se movió nada: %w",
		CoreMoveSubHalf:       "se movió la página a %s, pero no la carpeta de sus subpáginas: %w",
		CoreMoveReadBack:      "se movió la página, pero no se pudo volver a leer: %w",
		CoreBackslash:         `"\" no está permitido en un nombre`,
		CoreTrashMissing:      "no se encuentra la papelera: %w",
		CoreTrashFailed:       "no se pudo mover a la papelera: %w",
		CoreTrashFull:         "no se pudo mover a la papelera: hay demasiados elementos llamados %s",
		CoreCreateFailed:      "no se pudo crear la página: %w",
		CoreRelativeFolder:    "la ruta debe ser relativa a la carpeta",
		CoreRelativeWorkspace: "la ruta debe ser relativa a la carpeta de notas",
		CoreEmptySegment:      "la ruta tiene un nombre vacío",
		CoreDots:              `"." y ".." no están permitidos`,
		CorePageExists:        "aquí ya existe una página llamada %q",
		CoreFolderExists:      "aquí ya existe una carpeta llamada %q",
		CoreRenameFailed:      "no se pudo renombrar la página: %w",
		CoreRenameSubBack:     "no se pudieron renombrar las subpáginas ni devolver la nota a su sitio: %v: %w",
		CoreRenameSubNothing:  "no se pudieron renombrar las subpáginas, no se movió nada: %w",
		CoreRenameFolder:      "no se pudo renombrar la carpeta: %w",
		CoreDeleteFolder:      "no se pudo borrar la carpeta: %w",
		CoreDeletePage:        "no se pudo borrar la página: %w",
		CoreDeleteSubHalf:     "se borró la página, pero no la carpeta de sus subpáginas: %w",

		GraphNoNotes:                  "Aún no hay notas",
		GraphTooSmall:                 "El grafo necesita un panel más grande",
		GraphAllNotes:                 "Todas las notas",
		GraphLinks + one:              "%d enlace",
		GraphLinks + other:            "%d enlaces",
		GraphBacklinks + one:          "%d retroenlace",
		GraphBacklinks + other:        "%d retroenlaces",
		GraphNotes + one:              "%d nota",
		GraphNotes + other:            "%d notas",
		GraphSteps + one:              "%d paso",
		GraphSteps + other:            "%d pasos",
		GraphOrphans + one:            "%d huérfana",
		GraphOrphans + other:          "%d huérfanas",
		GraphUnresolved + one:         "%d sin resolver",
		GraphUnresolved + other:       "%d sin resolver",
		GraphMatches + one:            "%d coincidencia",
		GraphMatches + other:          "%d coincidencias",
		GraphOrphansHidden + one:      "%d huérfana oculta",
		GraphOrphansHidden + other:    "%d huérfanas ocultas",
		GraphUnresolvedHidden + one:   "%d sin resolver oculto",
		GraphUnresolvedHidden + other: "%d sin resolver ocultos",
		GraphFilterOn:                 " · filtro ",
		GraphCentre:                   "centrar",
		GraphBack:                     "volver",
		GraphReset:                    "inicio",
		GraphFilter:                   "filtrar",
		GraphOrphansBtn:               "huérfanas",
		GraphUnresBtn:                 "sin resolver",
		GraphTailFull:                 "  hjkl mover · enter abrir · clic: vista previa, otra vez: abrir · rueda: zoom · arrastrar: mover",
		GraphTailDrag:                 "  hjkl mover · enter abrir · arrastrar: mover",
		GraphTailOpen:                 "  hjkl mover · enter abrir",
		GraphTailKeys:                 "  hjkl · enter",
		GraphMapTailFull:              "  hjkl mover · n/N enlaces · enter abrir · clic: vista previa, otra vez: abrir · rueda: zoom · arrastrar: mover",
		GraphMapTailDrag:              "  hjkl mover · n/N enlaces · enter abrir · arrastrar: mover",
		GraphMapTailOpen:              "  hjkl mover · n/N enlaces · enter abrir",
		GraphPromptJump:               "   enter saltar · esc borrar",
	})
}

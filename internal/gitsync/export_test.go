package gitsync

// SetSyncHooks makes SyncVault call beforeApply right before it moves the
// branch with `merge --ff-only`, and beforePush right before it pushes, so
// a test can save a note or push from another device at that moment. The
// returned function removes the hooks.
func SetSyncHooks(beforeApply, beforePush func()) func() {
	testHookBeforeApply, testHookBeforePush = beforeApply, beforePush
	return func() { testHookBeforeApply, testHookBeforePush = nil, nil }
}

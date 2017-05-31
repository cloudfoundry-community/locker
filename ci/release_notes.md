# Improvements

A single unlock request for a key/requestor now results in all locks
for that key/requestor pair to be removed. A single key/requestor can still
unlock multiple times without error, but after the first one, its locks
have been released, and subsequent unlocks are no-ops. If another requestor
still retains a lock, the key is not removed.

This should make it easier to recover from failures, as you can now request
a lock, fail, request a lock again, succeed, release your lock, and not need
to intervene manually to clean up the original lock.

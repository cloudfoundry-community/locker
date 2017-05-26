# Revamped locking API

Gone are the concepts of locks on pools. In is the concept of a multitude of locks,
each lock can be locked by a single key at a time. However this key can lock the lock
multiple times, by multiple people who are allowed to share exclusivity on the lock.
If this happens, one must unlock said lock once for every person who locked it, and each
time that person locked it. Only then will the lock truly be free.

package main_test

import (
	"io/ioutil"

	. "github.com/cloudfoundry-community/locker"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Locker", func() {
	var locker Locker
	BeforeEach(func() {
		tmpFile, err := ioutil.TempFile("", "locker")
		Expect(err).ShouldNot(HaveOccurred())

		locker = Locker{LockConfig: tmpFile.Name()}
	})
	Context("when locking", func() {
		It("uses the key as requestor if requestor is not provided", func() {
			err := locker.Lock("myLock", "myKey", "")
			Expect(err).ShouldNot(HaveOccurred())
			lock, err := locker.GetLock("myLock")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(lock.Key).Should(Equal("myKey"))
			Expect(lock.LockedBy).Should(Equal(map[string]int{"myKey": 1}))
		})
		It("increments requestor when the key is correct", func() {
			err := locker.Lock("myLock", "myKey", "")
			Expect(err).ShouldNot(HaveOccurred())
			err = locker.Lock("myLock", "myKey", "")
			Expect(err).ShouldNot(HaveOccurred())
			lock, err := locker.GetLock("myLock")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(lock.Key).Should(Equal("myKey"))
			Expect(lock.LockedBy).Should(Equal(map[string]int{"myKey": 2}))
		})
		It("fails to lock, if the key doesn't match what's in the lock", func() {
			err := locker.Lock("myLock", "myKey", "")
			Expect(err).ShouldNot(HaveOccurred())
			err = locker.Lock("myLock", "newKey", "")
			Expect(err).Should(HaveOccurred())
			lock, err := locker.GetLock("myLock")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(lock.Key).Should(Equal("myKey"))
			Expect(lock.LockedBy).Should(Equal(map[string]int{"myKey": 1}))
		})
		It("is able to lock multiple requestors multiple times, if they all have the correct key", func() {
			err := locker.Lock("myLock", "myKey", "george")
			Expect(err).ShouldNot(HaveOccurred())
			err = locker.Lock("myLock", "myKey", "jane")
			Expect(err).ShouldNot(HaveOccurred())
			err = locker.Lock("myLock", "myKey", "george")
			Expect(err).ShouldNot(HaveOccurred())
			err = locker.Lock("myLock", "myKey", "jane")
			Expect(err).ShouldNot(HaveOccurred())

			lock, err := locker.GetLock("myLock")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(lock.Key).Should(Equal("myKey"))
			Expect(lock.LockedBy).Should(Equal(map[string]int{
				"jane":   2,
				"george": 2,
			}))
		})
	})
	Context("when unlocking", func() {
		It("uses the key as requestor if requestor is not provided", func() {
			err := locker.Lock("myLock", "myKey", "")
			Expect(err).ShouldNot(HaveOccurred())
			err = locker.Unlock("myLock", "myKey", "")
			Expect(err).ShouldNot(HaveOccurred())
			lock, err := locker.GetLock("myLock")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(lock.Key).Should(Equal(""))
			Expect(lock.LockedBy).Should(Equal(map[string]int{"myKey": 0}))
		})
		It("fully unlocks a key/requestor on a single unlock call", func() {
			err := locker.Lock("myLock", "myKey", "george")
			Expect(err).ShouldNot(HaveOccurred())
			err = locker.Lock("myLock", "myKey", "george")
			Expect(err).ShouldNot(HaveOccurred())
			lock, err := locker.GetLock("myLock")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(lock.LockedBy["george"]).Should(Equal(2))

			err = locker.Unlock("myLock", "myKey", "george")
			Expect(err).ShouldNot(HaveOccurred())
			lock, err = locker.GetLock("myLock")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(lock.LockedBy["george"]).Should(Equal(0))
		})
		It("does not error when decrementing a requestor that is not currently holding a lock", func() {
			err := locker.Lock("myLock", "myKey", "george")
			Expect(err).ShouldNot(HaveOccurred())
			err = locker.Unlock("myLock", "myKey", "jane")
			Expect(err).ShouldNot(HaveOccurred())

			lock, err := locker.GetLock("myLock")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(lock.LockedBy["george"]).Should(Equal(1))
			Expect(lock.LockedBy["jane"]).Should(Equal(0))
		})
		It("does not error when no one is holding a lock", func() {
			lock, err := locker.GetLock("myMissingLock")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(lock).Should(Equal(Lock{LockedBy: map[string]int{}}))
			err = locker.Unlock("myMissingLock", "anyKey", "whoCares?")
			Expect(err).ShouldNot(HaveOccurred())
			lock, err = locker.GetLock("myMissingLock")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(lock).Should(Equal(Lock{Key: "", LockedBy: map[string]int{"whoCares?": 0}}))
		})
		It("removes the key from the lock, if no remaining requestors hold locks", func() {
			err := locker.Lock("myLock", "myKey", "george")
			Expect(err).ShouldNot(HaveOccurred())
			lock, err := locker.GetLock("myLock")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(lock.Key).Should(Equal("myKey"))
			err = locker.Unlock("myLock", "myKey", "george")
			Expect(err).ShouldNot(HaveOccurred())
			lock, err = locker.GetLock("myLock")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(lock.Key).Should(Equal(""))
		})
		It("fails to unlock, if the key doesn't match what's in the lock", func() {
			err := locker.Lock("myLock", "myKey", "george")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(locker.Unlock("myLock", "wrongKey", "george")).ShouldNot(Succeed())
		})
		It("unlocking one of multiple requestors does not remove the key frmo the lock", func() {
			Expect(locker.Lock("myLock", "myKey", "george")).Should(Succeed())
			Expect(locker.Lock("myLock", "myKey", "george")).Should(Succeed())
			Expect(locker.Lock("myLock", "myKey", "jane")).Should(Succeed())
			Expect(locker.Lock("myLock", "myKey", "jane")).Should(Succeed())
			Expect(locker.Unlock("myLock", "myKey", "george")).Should(Succeed())
			lock, err := locker.GetLock("myLock")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(lock.LockedBy).Should(Equal(map[string]int{"jane": 2, "george": 0}))
		})
	})
	Context("when loading", func() {
		It("throws an error when the locker data could not be loaded", func() {
			locker.LockConfig = "DoesNotExist"
			_, err := locker.GetLocks()
			Expect(err).Should(HaveOccurred())
		})
		It("returns the on-disk locker data", func() {
			err := ioutil.WriteFile(locker.LockConfig, []byte(`
{
	"myLock": {
		"key": "myKey",
		"locked_by": {
			"george": 2,
			"jane": 15
		}
	}
}
`), 0644)
			Expect(err).ShouldNot(HaveOccurred())
			lock, err := locker.GetLock("myLock")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(lock.Key).Should(Equal("myKey"))
			Expect(lock.LockedBy).Should(Equal(map[string]int{"george": 2, "jane": 15}))
		})
	})
	Context("when saving", func() {
		It("throws an error when the locker data could not be saved", func() {
			locker.LockConfig = "Should/Fail/To/Save"
			err := locker.SaveLocks(LockerState{"myLock": Lock{Key: "myKey", LockedBy: map[string]int{"george": 2}}})
			Expect(err).Should(HaveOccurred())
		})
		It("saves locker data to disk", func() {
			err := locker.SaveLocks(LockerState{"myLock": Lock{Key: "myKey", LockedBy: map[string]int{"george": 2}}})
			Expect(err).ShouldNot(HaveOccurred())
			data, err := ioutil.ReadFile(locker.LockConfig)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(string(data)).Should(MatchJSON(`{"myLock":{"key":"myKey","locked_by":{"george":2}}}`))
		})
	})
	Context("when retrieving a lock", func() {
		It("returns the requested lock", func() {
			Expect(locker.Lock("myLock", "myKey", "george")).Should(Succeed())
			Expect(locker.Lock("otherLock", "otherKey", "jane")).Should(Succeed())
			lock, err := locker.GetLock("otherLock")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(lock).Should(Equal(Lock{Key: "otherKey", LockedBy: map[string]int{"jane": 1}}))
		})
		It("auto-vivifies the LockedBy map", func() {
			lock, err := locker.GetLock("notYetALock")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(lock.LockedBy).ShouldNot(BeNil())
			Expect(lock.LockedBy).Should(Equal(map[string]int{}))
		})
	})
})

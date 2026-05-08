package inbox

import (
	"log"

	"github.com/jacksonopp/openwaves/internal/activity"
	"github.com/jacksonopp/openwaves/internal/hls"
)

// TerminateStation clears the HLS store for username, propagates TerminateStream
// to all known followers, and calls onTerminate if non-nil.
// This is called by both the inbox handler and the admin handler.
func TerminateStation(username string, hlsStore *hls.Store, followerStore *FollowerStore, onTerminate func(string)) {
	ts := activity.TerminateStream{
		Type:   "TerminateStream",
		Actor:  username,
		Object: username,
	}
	hlsStore.Suspend(username)
	followers := followerStore.List(username)
	for _, f := range followers {
		sendActivity(f.InboxURL, ts)
	}
	if len(followers) > 0 {
		log.Printf("inbox: propagated TerminateStream to %d follower(s) of %s", len(followers), username)
	}
	if onTerminate != nil {
		onTerminate(username)
	}
}

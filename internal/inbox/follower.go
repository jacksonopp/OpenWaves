package inbox

// Follower represents a remote relay station that has sent a Follow activity.
type Follower struct {
	ActorURL string // URL of the remote station actor
	InboxURL string // URL of the remote station's inbox (for sending Accept/TerminateStream)
}

package features

// Available feature flags.
// TODO: these should be in a package of their own, but for now until we
// move around packages to have /internal/ namespaces leaving it here to avoid
// creating more top level packages just for this.
const (
	// MsgAttachGuide is support for resource guides as message attachments
	MsgAttachGuide = "msg-attach-guide"
	// OldRAFHomeCard is the old version of the refer-a-friend home card
	OldRAFHomeCard = "old-raf-home-card"
	// RAFHomeCard is the new version fot he refer-a-friend home card
	RAFHomeCard = "raf-home-card"
)

package backend

// BandwidthUpdater is implemented by backends that support runtime cap updates.
// Caps are expressed in megabits per second; non-positive values disable the cap.
type BandwidthUpdater interface {
	UpdateCaps(capReadMbps, capWriteMbps, capCombinedMbps float64)
}

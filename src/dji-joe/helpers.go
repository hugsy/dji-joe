package djijoe


func MessageTypeToString(MessageType int) string {
	switch MessageType {
	case TYPE_PROBE_REQUEST:
		return "ProbeRequest"

	case TYPE_BEACON:
		return "Beacon"

	default:
		Log.FatalF("Incorrect type %d", MessageType)
	}
	return ""
}



package domain

// FlattenPayloadContext extracts top-level fields and unwraps nested "attributes" maps
// into a single top-level map for evaluation context resolution.
func FlattenPayloadContext(payload map[string]interface{}) map[string]interface{} {
	ctx := make(map[string]interface{})
	if payload == nil {
		return ctx
	}

	for k, v := range payload {
		ctx[k] = v
		if k == "attributes" {
			if attrMap, ok := v.(map[string]interface{}); ok {
				for ak, av := range attrMap {
					ctx[ak] = av
				}
			} else if attrMapStr, ok := v.(map[string]string); ok {
				for ak, av := range attrMapStr {
					ctx[ak] = av
				}
			}
		}
	}
	return ctx
}

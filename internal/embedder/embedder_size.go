package embedder

// resolveEmbedContextSize returns the token budget for an embedding context.
// fromModel is the GGUF metadata context size (0 when unknown); a non-zero
// value wins so each model gets the window it was trained with.
func resolveEmbedContextSize(fromModel int) int {
	if fromModel > 0 {
		return fromModel
	}
	return 0
}

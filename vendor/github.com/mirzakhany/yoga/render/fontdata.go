package render

import _ "embed"

// InterTTF is the embedded UI sans-serif face (OFL).
//
//go:embed assets/Inter-Regular.ttf
var InterTTF []byte

// JetBrainsMonoTTF is the embedded monospace face for the code editor (OFL).
//
//go:embed assets/JetBrainsMono-Regular.ttf
var JetBrainsMonoTTF []byte

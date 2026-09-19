package render

import _ "embed"

// InterTTF is the embedded UI sans-serif face (OFL).
//
//go:embed assets/Inter-Regular.ttf
var InterTTF []byte

// InterSemiBoldTTF is the embedded UI semibold face for Weight 600 (OFL).
//
//go:embed assets/Inter-SemiBold.ttf
var InterSemiBoldTTF []byte

// JetBrainsMonoTTF is the embedded monospace face for the code editor (OFL).
//
//go:embed assets/JetBrainsMono-Regular.ttf
var JetBrainsMonoTTF []byte

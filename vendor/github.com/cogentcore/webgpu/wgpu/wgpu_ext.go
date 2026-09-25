package wgpu

const (
	// Buffer-Texture copies must have `TextureDataLayout.BytesPerRow` aligned to this number.
	//
	// This doesn't apply to `(*Queue).WriteTexture()`.
	CopyBytesPerRowAlignment = 256
	// An offset into the query resolve buffer has to be aligned to this.
	QueryResolveBufferAlignment = 256
	// Buffer to buffer copy as well as buffer clear offsets and sizes must be aligned to this number.
	CopyBufferAlignment = 4
	// Size to align mappings.
	MapAlignment = 8
	// Vertex buffer strides have to be aligned to this number.
	VertexStrideAlignment = 4
	// Alignment all push constants need
	PushConstantAlignment = 4
	// Maximum queries in a query set
	QuerySetMaxQueries = 8192
	// Size of a single piece of query data.
	QuerySize = 8
)

var (
	ColorTransparent = Color{0, 0, 0, 0}
	ColorBlack       = Color{0, 0, 0, 1}
	ColorWhite       = Color{1, 1, 1, 1}
	ColorRed         = Color{1, 0, 0, 1}
	ColorGreen       = Color{0, 1, 0, 1}
	ColorBlue        = Color{0, 0, 1, 1}

	BlendComponentReplace = BlendComponent{
		SrcFactor: BlendFactorOne,
		DstFactor: BlendFactorZero,
		Operation: BlendOperationAdd,
	}
	BlendComponentOver = BlendComponent{
		SrcFactor: BlendFactorOne,
		DstFactor: BlendFactorOneMinusSrcAlpha,
		Operation: BlendOperationAdd,
	}

	BlendStateReplace = BlendState{
		Color: BlendComponentReplace,
		Alpha: BlendComponentReplace,
	}
	BlendStateAlphaBlending = BlendState{
		Color: BlendComponent{
			SrcFactor: BlendFactorSrcAlpha,
			DstFactor: BlendFactorOneMinusSrcAlpha,
			Operation: BlendOperationAdd,
		},
		Alpha: BlendComponentOver,
	}
	BlendStatePremultipliedAlphaBlending = BlendState{
		Color: BlendComponentOver,
		Alpha: BlendComponentOver,
	}
)

func (v VertexFormat) Size() uint64 {
	switch v {
	case VertexFormatUint8x2,
		VertexFormatSint8x2,
		VertexFormatUnorm8x2,
		VertexFormatSnorm8x2:
		return 2

	case VertexFormatUint8x4,
		VertexFormatSint8x4,
		VertexFormatUnorm8x4,
		VertexFormatSnorm8x4,
		VertexFormatUint16x2,
		VertexFormatSint16x2,
		VertexFormatUnorm16x2,
		VertexFormatSnorm16x2,
		VertexFormatFloat16x2,
		VertexFormatFloat32,
		VertexFormatUint32,
		VertexFormatSint32:
		return 4

	case VertexFormatUint16x4,
		VertexFormatSint16x4,
		VertexFormatUnorm16x4,
		VertexFormatSnorm16x4,
		VertexFormatFloat16x4,
		VertexFormatFloat32x2,
		VertexFormatUint32x2,
		VertexFormatSint32x2:
		return 8

	case VertexFormatFloat32x3,
		VertexFormatUint32x3,
		VertexFormatSint32x3:
		return 12

	case VertexFormatFloat32x4,
		VertexFormatUint32x4,
		VertexFormatSint32x4:
		return 16

	default:
		return 0
	}
}

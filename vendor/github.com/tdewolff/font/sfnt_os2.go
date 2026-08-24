package font

import (
	"fmt"

	"github.com/tdewolff/parse/v2"
)

type os2Table struct {
	Version                 uint16
	XAvgCharWidth           int16
	UsWeightClass           uint16
	UsWidthClass            uint16
	FsType                  uint16
	YSubscriptXSize         int16
	YSubscriptYSize         int16
	YSubscriptXOffset       int16
	YSubscriptYOffset       int16
	YSuperscriptXSize       int16
	YSuperscriptYSize       int16
	YSuperscriptXOffset     int16
	YSuperscriptYOffset     int16
	YStrikeoutSize          int16
	YStrikeoutPosition      int16
	SFamilyClass            int16
	BFamilyType             uint8
	BSerifStyle             uint8
	BWeight                 uint8
	BProportion             uint8
	BContrast               uint8
	BStrokeVariation        uint8
	BArmStyle               uint8
	BLetterform             uint8
	BMidline                uint8
	BXHeight                uint8
	UlUnicodeRange1         uint32
	UlUnicodeRange2         uint32
	UlUnicodeRange3         uint32
	UlUnicodeRange4         uint32
	AchVendID               [4]byte
	FsSelection             uint16
	UsFirstCharIndex        uint16
	UsLastCharIndex         uint16
	STypoAscender           int16
	STypoDescender          int16
	STypoLineGap            int16
	UsWinAscent             uint16
	UsWinDescent            uint16
	UlCodePageRange1        uint32
	UlCodePageRange2        uint32
	SxHeight                int16
	SCapHeight              int16
	UsDefaultChar           uint16
	UsBreakChar             uint16
	UsMaxContent            uint16
	UsLowerOpticalPointSize uint16
	UsUpperOpticalPointSize uint16
}

func (sfnt *SFNT) parseOS2() error {
	b, ok := sfnt.Tables["OS/2"]
	if !ok {
		return fmt.Errorf("OS/2: missing table")
	} else if len(b) < 68 {
		return fmt.Errorf("OS/2: bad table")
	}

	r := parse.NewBinaryReaderBytes(b)
	sfnt.OS2 = &os2Table{}
	sfnt.OS2.Version = r.ReadUint16()
	if 5 < sfnt.OS2.Version {
		return fmt.Errorf("OS/2: bad version")
	} else if sfnt.OS2.Version == 0 && len(b) != 68 && len(b) != 78 ||
		sfnt.OS2.Version == 1 && len(b) != 86 ||
		2 <= sfnt.OS2.Version && sfnt.OS2.Version <= 4 && len(b) != 96 ||
		sfnt.OS2.Version == 5 && len(b) != 100 {
		return fmt.Errorf("OS/2: bad table")
	}
	sfnt.OS2.XAvgCharWidth = r.ReadInt16()
	sfnt.OS2.UsWeightClass = r.ReadUint16()
	sfnt.OS2.UsWidthClass = r.ReadUint16()
	sfnt.OS2.FsType = r.ReadUint16()
	sfnt.OS2.YSubscriptXSize = r.ReadInt16()
	sfnt.OS2.YSubscriptYSize = r.ReadInt16()
	sfnt.OS2.YSubscriptXOffset = r.ReadInt16()
	sfnt.OS2.YSubscriptYOffset = r.ReadInt16()
	sfnt.OS2.YSuperscriptXSize = r.ReadInt16()
	sfnt.OS2.YSuperscriptYSize = r.ReadInt16()
	sfnt.OS2.YSuperscriptXOffset = r.ReadInt16()
	sfnt.OS2.YSuperscriptYOffset = r.ReadInt16()
	sfnt.OS2.YStrikeoutSize = r.ReadInt16()
	sfnt.OS2.YStrikeoutPosition = r.ReadInt16()
	sfnt.OS2.SFamilyClass = r.ReadInt16()
	sfnt.OS2.BFamilyType = r.ReadUint8()
	sfnt.OS2.BSerifStyle = r.ReadUint8()
	sfnt.OS2.BWeight = r.ReadUint8()
	sfnt.OS2.BProportion = r.ReadUint8()
	sfnt.OS2.BContrast = r.ReadUint8()
	sfnt.OS2.BStrokeVariation = r.ReadUint8()
	sfnt.OS2.BArmStyle = r.ReadUint8()
	sfnt.OS2.BLetterform = r.ReadUint8()
	sfnt.OS2.BMidline = r.ReadUint8()
	sfnt.OS2.BXHeight = r.ReadUint8()
	sfnt.OS2.UlUnicodeRange1 = r.ReadUint32()
	sfnt.OS2.UlUnicodeRange2 = r.ReadUint32()
	sfnt.OS2.UlUnicodeRange3 = r.ReadUint32()
	sfnt.OS2.UlUnicodeRange4 = r.ReadUint32()
	copy(sfnt.OS2.AchVendID[:], r.ReadBytes(4))
	sfnt.OS2.FsSelection = r.ReadUint16()
	sfnt.OS2.UsFirstCharIndex = r.ReadUint16()
	sfnt.OS2.UsLastCharIndex = r.ReadUint16()
	if 78 <= len(b) {
		sfnt.OS2.STypoAscender = r.ReadInt16()
		sfnt.OS2.STypoDescender = r.ReadInt16()
		sfnt.OS2.STypoLineGap = r.ReadInt16()
		sfnt.OS2.UsWinAscent = r.ReadUint16()
		sfnt.OS2.UsWinDescent = r.ReadUint16()
	}
	if sfnt.OS2.Version == 0 {
		return nil
	}
	sfnt.OS2.UlCodePageRange1 = r.ReadUint32()
	sfnt.OS2.UlCodePageRange2 = r.ReadUint32()
	if sfnt.OS2.Version == 1 {
		return nil
	}
	sfnt.OS2.SxHeight = r.ReadInt16()
	sfnt.OS2.SCapHeight = r.ReadInt16()
	sfnt.OS2.UsDefaultChar = r.ReadUint16()
	sfnt.OS2.UsBreakChar = r.ReadUint16()
	sfnt.OS2.UsMaxContent = r.ReadUint16()
	if 2 <= sfnt.OS2.Version && sfnt.OS2.Version <= 4 {
		return nil
	}
	sfnt.OS2.UsLowerOpticalPointSize = r.ReadUint16()
	sfnt.OS2.UsUpperOpticalPointSize = r.ReadUint16()
	return nil
}

func (sfnt *SFNT) estimateOS2() {
	if sfnt.IsTrueType {
		contour, err := sfnt.Glyf.Contour(sfnt.GlyphIndex('x'))
		if err == nil {
			sfnt.OS2.SxHeight = contour.YMax
		}

		contour, err = sfnt.Glyf.Contour(sfnt.GlyphIndex('H'))
		if err == nil {
			sfnt.OS2.SCapHeight = contour.YMax
		}
	} else if sfnt.IsCFF {
		p := &bboxPather{}
		if err := sfnt.CFF.ToPath(p, sfnt.GlyphIndex('x'), 0, 0, 0, 1.0, NoHinting); err == nil {
			sfnt.OS2.SxHeight = int16(p.YMax)
		}

		p = &bboxPather{}
		if err := sfnt.CFF.ToPath(p, sfnt.GlyphIndex('H'), 0, 0, 0, 1.0, NoHinting); err == nil {
			sfnt.OS2.SCapHeight = int16(p.YMax)
		}
	}
}

func os2UlUnicodeRange(rs []rune) [4]uint32 {
	v := [4]uint32{0, 0, 0, 0}
	for _, r := range rs {
		if bit := os2UlUnicodeRangeBit(r); bit != -1 {
			i := int(bit / 32)
			v[3-i] |= 2 << (bit - i*32)
		}
		if 0x10000 <= r && r < 0x110000 {
			v[2] |= 2 << 25 // bit 57 (Non-Plane 0)
		}
	}
	return v
}

func os2UlUnicodeRangeBit(r rune) int {
	if r < 0x80 {
		return 0
	} else if r < 0x0100 {
		return 1
	} else if r < 0x0180 {
		return 2
	} else if r < 0x0250 {
		return 3
	} else if r < 0x02B0 {
		return 4
	} else if r < 0x0300 {
		return 5
	} else if r < 0x0370 {
		return 6
	} else if r < 0x0400 {
		return 7
	} else if r < 0x0500 {
		return 9
	} else if r < 0x0530 {
		return -1
	} else if r < 0x0590 {
		return 10
	} else if r < 0x0600 {
		return 11
	} else if r < 0x0700 {
		return 13
	} else if r < 0x0750 {
		return 71
	} else if r < 0x0780 {
		return -1
	} else if r < 0x07C0 {
		return 72
	} else if r < 0x0800 {
		return 14
	} else if r < 0x0900 {
		return -1
	} else if r < 0x0980 {
		return 15
	} else if r < 0x0A00 {
		return 16
	} else if r < 0x0A80 {
		return 17
	} else if r < 0x0B00 {
		return 18
	} else if r < 0x0B80 {
		return 19
	} else if r < 0x0C00 {
		return 20
	} else if r < 0x0C80 {
		return 21
	} else if r < 0x0D00 {
		return 22
	} else if r < 0x0D80 {
		return 23
	} else if r < 0x0E00 {
		return 73
	} else if r < 0x0E80 {
		return 24
	} else if r < 0x0F00 {
		return 25
	} else if r < 0x1000 {
		return 70
	} else if r < 0x10A0 {
		return 74
	} else if r < 0x1100 {
		return 26
	} else if r < 0x1200 {
		return 28
	} else if r < 0x1380 {
		return 75
	} else if r < 0x13A0 {
		return -1
	} else if r < 0x1400 {
		return 76
	} else if r < 0x1680 {
		return 77
	} else if r < 0x16A0 {
		return 78
	} else if r < 0x1700 {
		return 79
	} else if r < 0x1720 {
		return 84
	} else if r < 0x1780 {
		return -1
	} else if r < 0x1800 {
		return 80
	} else if r < 0x18B0 {
		return 81
	} else if r < 0x1900 {
		return -1
	} else if r < 0x1950 {
		return 93
	} else if r < 0x1980 {
		return 94
	} else if r < 0x19E0 {
		return 95
	} else if r < 0x1A00 {
		return -1
	} else if r < 0x1A20 {
		return 96
	} else if r < 0x1B00 {
		return -1
	} else if r < 0x1B80 {
		return 27
	} else if r < 0x1BC0 {
		return 112
	} else if r < 0x1C00 {
		return -1
	} else if r < 0x1C50 {
		return 113
	} else if r < 0x1C80 {
		return 114
	} else if r < 0x1E00 {
		return -1
	} else if r < 0x1F00 {
		return 29
	} else if r < 0x2000 {
		return 30
	} else if r < 0x2070 {
		return 31
	} else if r < 0x20A0 {
		return 32
	} else if r < 0x20D0 {
		return 33
	} else if r < 0x2100 {
		return 34
	} else if r < 0x2150 {
		return 35
	} else if r < 0x2190 {
		return 36
	} else if r < 0x2200 {
		return 37
	} else if r < 0x2300 {
		return 38
	} else if r < 0x2400 {
		return 39
	} else if r < 0x2440 {
		return 40
	} else if r < 0x2460 {
		return 41
	} else if r < 0x2500 {
		return 42
	} else if r < 0x2580 {
		return 43
	} else if r < 0x25A0 {
		return 44
	} else if r < 0x2600 {
		return 45
	} else if r < 0x2700 {
		return 46
	} else if r < 0x27C0 {
		return 47
	} else if r < 0x2800 {
		return -1
	} else if r < 0x2900 {
		return 82
	} else if r < 0x2C00 {
		return -1
	} else if r < 0x2C60 {
		return 97
	} else if r < 0x2C80 {
		return -1
	} else if r < 0x2D00 {
		return 8
	} else if r < 0x2D30 {
		return -1
	} else if r < 0x2D80 {
		return 98
	} else if r < 0x3000 {
		return -1
	} else if r < 0x3040 {
		return 48
	} else if r < 0x30A0 {
		return 49
	} else if r < 0x3100 {
		return 50
	} else if r < 0x3130 {
		return 51
	} else if r < 0x3190 {
		return 52
	} else if r < 0x3200 {
		return -1
	} else if r < 0x3300 {
		return 54
	} else if r < 0x3400 {
		return 55
	} else if r < 0x31C0 {
		return -1
	} else if r < 0x31F0 {
		return 61
	} else if r < 0x4DC0 {
		return -1
	} else if r < 0x4E00 {
		return 99
	} else if r < 0xA000 {
		return 59
	} else if r < 0xA490 {
		return 83
	} else if r < 0xA500 {
		return -1
	} else if r < 0xA640 {
		return 12
	} else if r < 0xA800 {
		return -1
	} else if r < 0xA830 {
		return 100
	} else if r < 0xA840 {
		return -1
	} else if r < 0xA880 {
		return 53
	} else if r < 0xA8E0 {
		return 115
	} else if r < 0xA900 {
		return -1
	} else if r < 0xA930 {
		return 116
	} else if r < 0xA960 {
		return 117
	} else if r < 0xAA00 {
		return -1
	} else if r < 0xAA60 {
		return 118
	} else if r < 0xAC00 {
		return -1
	} else if r < 0xD7AF {
		return 56
	} else if r < 0xE00 {
		return -1
	} else if r < 0xF900 {
		return 60
	} else if r < 0xFB00 {
		return -1
	} else if r < 0xFB50 {
		return 62
	} else if r < 0xFE00 {
		return 63
	} else if r < 0xFE10 {
		return 91
	} else if r < 0xFE20 {
		return 65
	} else if r < 0xFE30 {
		return 64
	} else if r < 0xFE50 {
		return -1
	} else if r < 0xFE70 {
		return 66
	} else if r < 0xFF00 {
		return 67
	} else if r < 0xFFF0 {
		return 68
	} else if r < 0x10000 {
		return 69
	} else if r < 0x10080 {
		return 101
	} else if r < 0x10140 {
		return -1
	} else if r < 0x10190 {
		return 102
	} else if r < 0x101D0 {
		return 119
	} else if r < 0x10200 {
		return 120
	} else if r < 0x102A0 {
		return -1
	} else if r < 0x102E0 {
		return 121
	} else if r < 0x10300 {
		return -1
	} else if r < 0x10330 {
		return 85
	} else if r < 0x10350 {
		return 86
	} else if r < 0x10380 {
		return -1
	} else if r < 0x103A0 {
		return 103
	} else if r < 0x103E0 {
		return 104
	} else if r < 0x10400 {
		return -1
	} else if r < 0x10450 {
		return 87
	} else if r < 0x10480 {
		return 105
	} else if r < 0x104B0 {
		return 106
	} else if r < 0x10800 {
		return -1
	} else if r < 0x10840 {
		return 107
	} else if r < 0x10A00 {
		return -1
	} else if r < 0x10A60 {
		return 108
	} else if r < 0x12000 {
		return -1
	} else if r < 0x12400 {
		return 110
	} else if r < 0x1D000 {
		return -1
	} else if r < 0x1D100 {
		return 88
	} else if r < 0x1D300 {
		return -1
	} else if r < 0x1D360 {
		return 109
	} else if r < 0x1D380 {
		return 111
	} else if r < 0x1D400 {
		return -1
	} else if r < 0x1D800 {
		return 89
	} else if r < 0x1F030 {
		return -1
	} else if r < 0x1F0A0 {
		return 122
	} else if r < 0xE0000 {
		return -1
	} else if r < 0xE0080 {
		return 92
	} else if r < 0xF0000 {
		return -1
	} else if r < 0xFFFFE {
		return 90
	}
	return -1
}

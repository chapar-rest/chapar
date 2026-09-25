package font

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/tdewolff/parse/v2"
)

// ParseEOT parses the EOT font format and returns its contained SFNT font format (TTF or OTF). See https://www.w3.org/Submission/EOT/
func ParseEOT(b []byte) ([]byte, error) {
	r := parse.NewBinaryReaderBytes(b)
	r.ByteOrder = binary.LittleEndian
	_ = r.ReadUint32()             // EOTSize
	fontDataSize := r.ReadUint32() // FontDataSize
	version := r.ReadUint32()      // Version
	if version != 0x00010000 && version != 0x00020001 && version != 0x00020002 {
		return nil, fmt.Errorf("unsupported version")
	}
	flags := r.ReadUint32()       // Flags
	_ = r.ReadBytes(10)           // FontPANOSE
	_ = r.ReadUint8()             // Charset
	_ = r.ReadUint8()             // Italic
	_ = r.ReadUint32()            // Weight
	_ = r.ReadUint16()            // fsType
	magicNumber := r.ReadUint16() // MagicNumber
	if magicNumber != 0x504C {
		return nil, fmt.Errorf("invalid magic number")
	}
	_ = r.ReadBytes(24) // Unicode and CodePage ranges
	checkSumAdjustment := r.ReadUint32()
	_ = r.ReadBytes(16) // Reserved
	_ = r.ReadUint16()  // Padding1

	familyNameSize := r.ReadUint16()       // FamilyNameSize
	_ = r.ReadBytes(int64(familyNameSize)) // FamilyName
	_ = r.ReadUint16()                     // Padding2

	styleNameSize := r.ReadUint16()       // StyleNameSize
	_ = r.ReadBytes(int64(styleNameSize)) // Stylename
	_ = r.ReadUint16()                    // Padding3

	versionNameSize := r.ReadUint16()       // VersionNameSize
	_ = r.ReadBytes(int64(versionNameSize)) // VersionName
	_ = r.ReadUint16()                      // Padding4

	fullNameSize := r.ReadUint16()       // FullNameSize
	_ = r.ReadBytes(int64(fullNameSize)) // FullName

	if version == 0x00020001 || version == 0x00020002 {
		_ = r.ReadUint16()                     // Padding5
		rootStringSize := r.ReadUint16()       // RootStringSize
		_ = r.ReadBytes(int64(rootStringSize)) // RootString
	}
	if version == 0x00020002 {
		_ = r.ReadUint32()                    // RootStringCheckSum
		_ = r.ReadUint32()                    // EUDCCodePage
		_ = r.ReadUint16()                    // Padding6
		signatureSize := r.ReadUint16()       // SignatureSize
		_ = r.ReadBytes(int64(signatureSize)) // Signature
		_ = r.ReadUint32()                    // EUDCFlags
		eudcFontSize := r.ReadUint32()        // EUDCFontSize
		_ = r.ReadBytes(int64(eudcFontSize))  // EUDCFontData
	}

	fontData := r.ReadBytes(int64(fontDataSize))
	if err := r.Err(); err == io.EOF {
		return nil, ErrInvalidFontData
	} else if err != nil {
		return nil, err
	}

	isCompressed := (flags & 0x00000004) != 0
	isXORed := (flags & 0x10000000) != 0

	if isXORed {
		for i := 0; i < len(fontData); i++ {
			fontData[i] ^= 0x50
		}
	}

	if isCompressed {
		// TODO: (EOT) see https://www.w3.org/Submission/MTX/
		return nil, fmt.Errorf("EOT compression not supported")
	}

	_ = checkSumAdjustment
	// TODO: (EOT) verify or recalculate master checksum
	//checksum := 0xB1B0AFBA - calcChecksum(w.Bytes())
	//if checkSumAdjustment != checksum {
	//return nil, 0, fmt.Errorf("bad checksum")
	//}

	// replace overal checksum in head table
	//buf := w.Bytes()
	//binary.BigEndian.PutUint32(buf[iCheckSumAdjustment:], checkSumAdjustment)
	return fontData, nil
}

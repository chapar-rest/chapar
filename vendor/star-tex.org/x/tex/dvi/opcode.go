// Copyright ©2021 The star-tex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dvi

import (
	"fmt"
)

const (
	dviEOF = 223

	dviVersion = 2
)

type opCode uint8

const (
	opSetChar000 opCode = iota // typeset a character and move right
	opSetChar001               // typeset a character and move right
	opSetChar002               // typeset a character and move right
	opSetChar003               // typeset a character and move right
	opSetChar004               // typeset a character and move right
	opSetChar005               // typeset a character and move right
	opSetChar006               // typeset a character and move right
	opSetChar007               // typeset a character and move right
	opSetChar008               // typeset a character and move right
	opSetChar009               // typeset a character and move right
	opSetChar010               // typeset a character and move right
	opSetChar011               // typeset a character and move right
	opSetChar012               // typeset a character and move right
	opSetChar013               // typeset a character and move right
	opSetChar014               // typeset a character and move right
	opSetChar015               // typeset a character and move right
	opSetChar016               // typeset a character and move right
	opSetChar017               // typeset a character and move right
	opSetChar018               // typeset a character and move right
	opSetChar019               // typeset a character and move right
	opSetChar020               // typeset a character and move right
	opSetChar021               // typeset a character and move right
	opSetChar022               // typeset a character and move right
	opSetChar023               // typeset a character and move right
	opSetChar024               // typeset a character and move right
	opSetChar025               // typeset a character and move right
	opSetChar026               // typeset a character and move right
	opSetChar027               // typeset a character and move right
	opSetChar028               // typeset a character and move right
	opSetChar029               // typeset a character and move right
	opSetChar030               // typeset a character and move right
	opSetChar031               // typeset a character and move right
	opSetChar032               // typeset a character and move right
	opSetChar033               // typeset a character and move right
	opSetChar034               // typeset a character and move right
	opSetChar035               // typeset a character and move right
	opSetChar036               // typeset a character and move right
	opSetChar037               // typeset a character and move right
	opSetChar038               // typeset a character and move right
	opSetChar039               // typeset a character and move right
	opSetChar040               // typeset a character and move right
	opSetChar041               // typeset a character and move right
	opSetChar042               // typeset a character and move right
	opSetChar043               // typeset a character and move right
	opSetChar044               // typeset a character and move right
	opSetChar045               // typeset a character and move right
	opSetChar046               // typeset a character and move right
	opSetChar047               // typeset a character and move right
	opSetChar048               // typeset a character and move right
	opSetChar049               // typeset a character and move right
	opSetChar050               // typeset a character and move right
	opSetChar051               // typeset a character and move right
	opSetChar052               // typeset a character and move right
	opSetChar053               // typeset a character and move right
	opSetChar054               // typeset a character and move right
	opSetChar055               // typeset a character and move right
	opSetChar056               // typeset a character and move right
	opSetChar057               // typeset a character and move right
	opSetChar058               // typeset a character and move right
	opSetChar059               // typeset a character and move right
	opSetChar060               // typeset a character and move right
	opSetChar061               // typeset a character and move right
	opSetChar062               // typeset a character and move right
	opSetChar063               // typeset a character and move right
	opSetChar064               // typeset a character and move right
	opSetChar065               // typeset a character and move right
	opSetChar066               // typeset a character and move right
	opSetChar067               // typeset a character and move right
	opSetChar068               // typeset a character and move right
	opSetChar069               // typeset a character and move right
	opSetChar070               // typeset a character and move right
	opSetChar071               // typeset a character and move right
	opSetChar072               // typeset a character and move right
	opSetChar073               // typeset a character and move right
	opSetChar074               // typeset a character and move right
	opSetChar075               // typeset a character and move right
	opSetChar076               // typeset a character and move right
	opSetChar077               // typeset a character and move right
	opSetChar078               // typeset a character and move right
	opSetChar079               // typeset a character and move right
	opSetChar080               // typeset a character and move right
	opSetChar081               // typeset a character and move right
	opSetChar082               // typeset a character and move right
	opSetChar083               // typeset a character and move right
	opSetChar084               // typeset a character and move right
	opSetChar085               // typeset a character and move right
	opSetChar086               // typeset a character and move right
	opSetChar087               // typeset a character and move right
	opSetChar088               // typeset a character and move right
	opSetChar089               // typeset a character and move right
	opSetChar090               // typeset a character and move right
	opSetChar091               // typeset a character and move right
	opSetChar092               // typeset a character and move right
	opSetChar093               // typeset a character and move right
	opSetChar094               // typeset a character and move right
	opSetChar095               // typeset a character and move right
	opSetChar096               // typeset a character and move right
	opSetChar097               // typeset a character and move right
	opSetChar098               // typeset a character and move right
	opSetChar099               // typeset a character and move right
	opSetChar100               // typeset a character and move right
	opSetChar101               // typeset a character and move right
	opSetChar102               // typeset a character and move right
	opSetChar103               // typeset a character and move right
	opSetChar104               // typeset a character and move right
	opSetChar105               // typeset a character and move right
	opSetChar106               // typeset a character and move right
	opSetChar107               // typeset a character and move right
	opSetChar108               // typeset a character and move right
	opSetChar109               // typeset a character and move right
	opSetChar110               // typeset a character and move right
	opSetChar111               // typeset a character and move right
	opSetChar112               // typeset a character and move right
	opSetChar113               // typeset a character and move right
	opSetChar114               // typeset a character and move right
	opSetChar115               // typeset a character and move right
	opSetChar116               // typeset a character and move right
	opSetChar117               // typeset a character and move right
	opSetChar118               // typeset a character and move right
	opSetChar119               // typeset a character and move right
	opSetChar120               // typeset a character and move right
	opSetChar121               // typeset a character and move right
	opSetChar122               // typeset a character and move right
	opSetChar123               // typeset a character and move right
	opSetChar124               // typeset a character and move right
	opSetChar125               // typeset a character and move right
	opSetChar126               // typeset a character and move right
	opSetChar127               // typeset a character and move right
	opSet1                     // typeset a character and move right
	opSet2                     // typeset a character and move right
	opSet3                     // typeset a character and move right
	opSet4                     // typeset a character and move right
	opSetRule                  // typeset a rule and move right
	opPut1                     // typeset a character
	opPut2                     // typeset a character
	opPut3                     // typeset a character
	opPut4                     // typeset a character
	opPutRule                  // typeset a rule
	opNOP                      // no operation
	opBOP                      // beginning of page
	opEOP                      // ending of page
	opPush                     // save the current positions
	opPop                      // restore previous positions
	opRight1                   // move right
	opRight2                   // move right
	opRight3                   // move right
	opRight4                   // move right
	opW0                       // move right by w
	opW1                       // move right and set w
	opW2                       // move right and set w
	opW3                       // move right and set w
	opW4                       // move right and set w
	opX0                       // move right by x
	opX1                       // move right and set x
	opX2                       // move right and set x
	opX3                       // move right and set x
	opX4                       // move right and set x
	opDown1                    // move down
	opDown2                    // move down
	opDown3                    // move down
	opDown4                    // move down
	opY0                       // move down by y
	opY1                       // move down and set y
	opY2                       // move down and set y
	opY3                       // move down and set y
	opY4                       // move down and set y
	opZ0                       // move down by z
	opZ1                       // move down and set z
	opZ2                       // move down and set z
	opZ3                       // move down and set z
	opZ4                       // move down and set z
	opFntNum00                 // set current font to i
	opFntNum01                 // set current font to i
	opFntNum02                 // set current font to i
	opFntNum03                 // set current font to i
	opFntNum04                 // set current font to i
	opFntNum05                 // set current font to i
	opFntNum06                 // set current font to i
	opFntNum07                 // set current font to i
	opFntNum08                 // set current font to i
	opFntNum09                 // set current font to i
	opFntNum10                 // set current font to i
	opFntNum11                 // set current font to i
	opFntNum12                 // set current font to i
	opFntNum13                 // set current font to i
	opFntNum14                 // set current font to i
	opFntNum15                 // set current font to i
	opFntNum16                 // set current font to i
	opFntNum17                 // set current font to i
	opFntNum18                 // set current font to i
	opFntNum19                 // set current font to i
	opFntNum20                 // set current font to i
	opFntNum21                 // set current font to i
	opFntNum22                 // set current font to i
	opFntNum23                 // set current font to i
	opFntNum24                 // set current font to i
	opFntNum25                 // set current font to i
	opFntNum26                 // set current font to i
	opFntNum27                 // set current font to i
	opFntNum28                 // set current font to i
	opFntNum29                 // set current font to i
	opFntNum30                 // set current font to i
	opFntNum31                 // set current font to i
	opFntNum32                 // set current font to i
	opFntNum33                 // set current font to i
	opFntNum34                 // set current font to i
	opFntNum35                 // set current font to i
	opFntNum36                 // set current font to i
	opFntNum37                 // set current font to i
	opFntNum38                 // set current font to i
	opFntNum39                 // set current font to i
	opFntNum40                 // set current font to i
	opFntNum41                 // set current font to i
	opFntNum42                 // set current font to i
	opFntNum43                 // set current font to i
	opFntNum44                 // set current font to i
	opFntNum45                 // set current font to i
	opFntNum46                 // set current font to i
	opFntNum47                 // set current font to i
	opFntNum48                 // set current font to i
	opFntNum49                 // set current font to i
	opFntNum50                 // set current font to i
	opFntNum51                 // set current font to i
	opFntNum52                 // set current font to i
	opFntNum53                 // set current font to i
	opFntNum54                 // set current font to i
	opFntNum55                 // set current font to i
	opFntNum56                 // set current font to i
	opFntNum57                 // set current font to i
	opFntNum58                 // set current font to i
	opFntNum59                 // set current font to i
	opFntNum60                 // set current font to i
	opFntNum61                 // set current font to i
	opFntNum62                 // set current font to i
	opFntNum63                 // set current font to i
	opFnt1                     // set current font
	opFnt2                     // set current font
	opFnt3                     // set current font
	opFnt4                     // set current font
	opXXX1                     // extension to DVI primitives
	opXXX2                     // extension to DVI primitives
	opXXX3                     // extension to DVI primitives
	opXXX4                     // extension to DVI primitives
	opFntDef1                  // define the meaning of a font number
	opFntDef2                  // define the meaning of a font number
	opFntDef3                  // define the meaning of a font number
	opFntDef4                  // define the meaning of a font number
	opPre                      // preambule
	opPost                     // postamble beginning
	opPostPost                 // postamble ending
)

func (op opCode) cmd() Cmd {
	switch op {
	case opSetChar000, opSetChar001, opSetChar002, opSetChar003, opSetChar004,
		opSetChar005, opSetChar006, opSetChar007, opSetChar008, opSetChar009,
		opSetChar010, opSetChar011, opSetChar012, opSetChar013, opSetChar014,
		opSetChar015, opSetChar016, opSetChar017, opSetChar018, opSetChar019,
		opSetChar020, opSetChar021, opSetChar022, opSetChar023, opSetChar024,
		opSetChar025, opSetChar026, opSetChar027, opSetChar028, opSetChar029,
		opSetChar030, opSetChar031, opSetChar032, opSetChar033, opSetChar034,
		opSetChar035, opSetChar036, opSetChar037, opSetChar038, opSetChar039,
		opSetChar040, opSetChar041, opSetChar042, opSetChar043, opSetChar044,
		opSetChar045, opSetChar046, opSetChar047, opSetChar048, opSetChar049,
		opSetChar050, opSetChar051, opSetChar052, opSetChar053, opSetChar054,
		opSetChar055, opSetChar056, opSetChar057, opSetChar058, opSetChar059,
		opSetChar060, opSetChar061, opSetChar062, opSetChar063, opSetChar064,
		opSetChar065, opSetChar066, opSetChar067, opSetChar068, opSetChar069,
		opSetChar070, opSetChar071, opSetChar072, opSetChar073, opSetChar074,
		opSetChar075, opSetChar076, opSetChar077, opSetChar078, opSetChar079,
		opSetChar080, opSetChar081, opSetChar082, opSetChar083, opSetChar084,
		opSetChar085, opSetChar086, opSetChar087, opSetChar088, opSetChar089,
		opSetChar090, opSetChar091, opSetChar092, opSetChar093, opSetChar094,
		opSetChar095, opSetChar096, opSetChar097, opSetChar098, opSetChar099,
		opSetChar100, opSetChar101, opSetChar102, opSetChar103, opSetChar104,
		opSetChar105, opSetChar106, opSetChar107, opSetChar108, opSetChar109,
		opSetChar110, opSetChar111, opSetChar112, opSetChar113, opSetChar114,
		opSetChar115, opSetChar116, opSetChar117, opSetChar118, opSetChar119,
		opSetChar120, opSetChar121, opSetChar122, opSetChar123, opSetChar124,
		opSetChar125, opSetChar126, opSetChar127:
		return &CmdSetChar{}
	case opSet1:
		return &CmdSet1{}
	case opSet2:
		return &CmdSet2{}
	case opSet3:
		return &CmdSet3{}
	case opSet4:
		return &CmdSet4{}
	case opSetRule:
		return &CmdSetRule{}
	case opPut1:
		return &CmdPut1{}
	case opPut2:
		return &CmdPut2{}
	case opPut3:
		return &CmdPut3{}
	case opPut4:
		return &CmdPut4{}
	case opPutRule:
		return &CmdPutRule{}
	case opNOP:
		return &CmdNOP{}
	case opBOP:
		return &CmdBOP{}
	case opEOP:
		return &CmdEOP{}
	case opPush:
		return &CmdPush{}
	case opPop:
		return &CmdPop{}
	case opRight1:
		return &CmdRight1{}
	case opRight2:
		return &CmdRight2{}
	case opRight3:
		return &CmdRight3{}
	case opRight4:
		return &CmdRight4{}
	case opW0:
		return &CmdW0{}
	case opW1:
		return &CmdW1{}
	case opW2:
		return &CmdW2{}
	case opW3:
		return &CmdW3{}
	case opW4:
		return &CmdW4{}
	case opX0:
		return &CmdX0{}
	case opX1:
		return &CmdX1{}
	case opX2:
		return &CmdX2{}
	case opX3:
		return &CmdX3{}
	case opX4:
		return &CmdX4{}
	case opDown1:
		return &CmdDown1{}
	case opDown2:
		return &CmdDown2{}
	case opDown3:
		return &CmdDown3{}
	case opDown4:
		return &CmdDown4{}
	case opY0:
		return &CmdY0{}
	case opY1:
		return &CmdY1{}
	case opY2:
		return &CmdY2{}
	case opY3:
		return &CmdY3{}
	case opY4:
		return &CmdY4{}
	case opZ0:
		return &CmdZ0{}
	case opZ1:
		return &CmdZ1{}
	case opZ2:
		return &CmdZ2{}
	case opZ3:
		return &CmdZ3{}
	case opZ4:
		return &CmdZ4{}
	case opFntNum00, opFntNum01, opFntNum02, opFntNum03, opFntNum04,
		opFntNum05, opFntNum06, opFntNum07, opFntNum08, opFntNum09,
		opFntNum10, opFntNum11, opFntNum12, opFntNum13, opFntNum14,
		opFntNum15, opFntNum16, opFntNum17, opFntNum18, opFntNum19,
		opFntNum20, opFntNum21, opFntNum22, opFntNum23, opFntNum24,
		opFntNum25, opFntNum26, opFntNum27, opFntNum28, opFntNum29,
		opFntNum30, opFntNum31, opFntNum32, opFntNum33, opFntNum34,
		opFntNum35, opFntNum36, opFntNum37, opFntNum38, opFntNum39,
		opFntNum40, opFntNum41, opFntNum42, opFntNum43, opFntNum44,
		opFntNum45, opFntNum46, opFntNum47, opFntNum48, opFntNum49,
		opFntNum50, opFntNum51, opFntNum52, opFntNum53, opFntNum54,
		opFntNum55, opFntNum56, opFntNum57, opFntNum58, opFntNum59,
		opFntNum60, opFntNum61, opFntNum62, opFntNum63:
		return &CmdFntNum{}

	case opFnt1:
		return &CmdFnt1{}
	case opFnt2:
		return &CmdFnt2{}
	case opFnt3:
		return &CmdFnt3{}
	case opFnt4:
		return &CmdFnt4{}
	case opXXX1:
		return &CmdXXX1{}
	case opXXX2:
		return &CmdXXX2{}
	case opXXX3:
		return &CmdXXX3{}
	case opXXX4:
		return &CmdXXX4{}
	case opFntDef1:
		return &CmdFntDef1{}
	case opFntDef2:
		return &CmdFntDef2{}
	case opFntDef3:
		return &CmdFntDef3{}
	case opFntDef4:
		return &CmdFntDef4{}
	case opPre:
		return &CmdPre{}
	case opPost:
		return &CmdPost{}
	case opPostPost:
		return &CmdPostPost{}
	default:
		panic(fmt.Errorf("dvi: unknown opcode 0x%x (%d)", op, op))
	}
}

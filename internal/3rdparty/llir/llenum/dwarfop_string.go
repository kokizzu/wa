// Code generated by "stringer -linecomment -type DwarfOp"; DO NOT EDIT.

package llenum

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[DwarfOpAddr-3]
	_ = x[DwarfOpDeref-6]
	_ = x[DwarfOpConst1u-8]
	_ = x[DwarfOpConst1s-9]
	_ = x[DwarfOpConst2u-10]
	_ = x[DwarfOpConst2s-11]
	_ = x[DwarfOpConst4u-12]
	_ = x[DwarfOpConst4s-13]
	_ = x[DwarfOpConst8u-14]
	_ = x[DwarfOpConst8s-15]
	_ = x[DwarfOpConstu-16]
	_ = x[DwarfOpConsts-17]
	_ = x[DwarfOpDup-18]
	_ = x[DwarfOpDrop-19]
	_ = x[DwarfOpOver-20]
	_ = x[DwarfOpPick-21]
	_ = x[DwarfOpSwap-22]
	_ = x[DwarfOpRot-23]
	_ = x[DwarfOpXderef-24]
	_ = x[DwarfOpAbs-25]
	_ = x[DwarfOpAnd-26]
	_ = x[DwarfOpDiv-27]
	_ = x[DwarfOpMinus-28]
	_ = x[DwarfOpMod-29]
	_ = x[DwarfOpMul-30]
	_ = x[DwarfOpNeg-31]
	_ = x[DwarfOpNot-32]
	_ = x[DwarfOpOr-33]
	_ = x[DwarfOpPlus-34]
	_ = x[DwarfOpPlusUconst-35]
	_ = x[DwarfOpShl-36]
	_ = x[DwarfOpShr-37]
	_ = x[DwarfOpShra-38]
	_ = x[DwarfOpXor-39]
	_ = x[DwarfOpBra-40]
	_ = x[DwarfOpEq-41]
	_ = x[DwarfOpGe-42]
	_ = x[DwarfOpGt-43]
	_ = x[DwarfOpLe-44]
	_ = x[DwarfOpLt-45]
	_ = x[DwarfOpNe-46]
	_ = x[DwarfOpSkip-47]
	_ = x[DwarfOpLit0-48]
	_ = x[DwarfOpLit1-49]
	_ = x[DwarfOpLit2-50]
	_ = x[DwarfOpLit3-51]
	_ = x[DwarfOpLit4-52]
	_ = x[DwarfOpLit5-53]
	_ = x[DwarfOpLit6-54]
	_ = x[DwarfOpLit7-55]
	_ = x[DwarfOpLit8-56]
	_ = x[DwarfOpLit9-57]
	_ = x[DwarfOpLit10-58]
	_ = x[DwarfOpLit11-59]
	_ = x[DwarfOpLit12-60]
	_ = x[DwarfOpLit13-61]
	_ = x[DwarfOpLit14-62]
	_ = x[DwarfOpLit15-63]
	_ = x[DwarfOpLit16-64]
	_ = x[DwarfOpLit17-65]
	_ = x[DwarfOpLit18-66]
	_ = x[DwarfOpLit19-67]
	_ = x[DwarfOpLit20-68]
	_ = x[DwarfOpLit21-69]
	_ = x[DwarfOpLit22-70]
	_ = x[DwarfOpLit23-71]
	_ = x[DwarfOpLit24-72]
	_ = x[DwarfOpLit25-73]
	_ = x[DwarfOpLit26-74]
	_ = x[DwarfOpLit27-75]
	_ = x[DwarfOpLit28-76]
	_ = x[DwarfOpLit29-77]
	_ = x[DwarfOpLit30-78]
	_ = x[DwarfOpLit31-79]
	_ = x[DwarfOpReg0-80]
	_ = x[DwarfOpReg1-81]
	_ = x[DwarfOpReg2-82]
	_ = x[DwarfOpReg3-83]
	_ = x[DwarfOpReg4-84]
	_ = x[DwarfOpReg5-85]
	_ = x[DwarfOpReg6-86]
	_ = x[DwarfOpReg7-87]
	_ = x[DwarfOpReg8-88]
	_ = x[DwarfOpReg9-89]
	_ = x[DwarfOpReg10-90]
	_ = x[DwarfOpReg11-91]
	_ = x[DwarfOpReg12-92]
	_ = x[DwarfOpReg13-93]
	_ = x[DwarfOpReg14-94]
	_ = x[DwarfOpReg15-95]
	_ = x[DwarfOpReg16-96]
	_ = x[DwarfOpReg17-97]
	_ = x[DwarfOpReg18-98]
	_ = x[DwarfOpReg19-99]
	_ = x[DwarfOpReg20-100]
	_ = x[DwarfOpReg21-101]
	_ = x[DwarfOpReg22-102]
	_ = x[DwarfOpReg23-103]
	_ = x[DwarfOpReg24-104]
	_ = x[DwarfOpReg25-105]
	_ = x[DwarfOpReg26-106]
	_ = x[DwarfOpReg27-107]
	_ = x[DwarfOpReg28-108]
	_ = x[DwarfOpReg29-109]
	_ = x[DwarfOpReg30-110]
	_ = x[DwarfOpReg31-111]
	_ = x[DwarfOpBreg0-112]
	_ = x[DwarfOpBreg1-113]
	_ = x[DwarfOpBreg2-114]
	_ = x[DwarfOpBreg3-115]
	_ = x[DwarfOpBreg4-116]
	_ = x[DwarfOpBreg5-117]
	_ = x[DwarfOpBreg6-118]
	_ = x[DwarfOpBreg7-119]
	_ = x[DwarfOpBreg8-120]
	_ = x[DwarfOpBreg9-121]
	_ = x[DwarfOpBreg10-122]
	_ = x[DwarfOpBreg11-123]
	_ = x[DwarfOpBreg12-124]
	_ = x[DwarfOpBreg13-125]
	_ = x[DwarfOpBreg14-126]
	_ = x[DwarfOpBreg15-127]
	_ = x[DwarfOpBreg16-128]
	_ = x[DwarfOpBreg17-129]
	_ = x[DwarfOpBreg18-130]
	_ = x[DwarfOpBreg19-131]
	_ = x[DwarfOpBreg20-132]
	_ = x[DwarfOpBreg21-133]
	_ = x[DwarfOpBreg22-134]
	_ = x[DwarfOpBreg23-135]
	_ = x[DwarfOpBreg24-136]
	_ = x[DwarfOpBreg25-137]
	_ = x[DwarfOpBreg26-138]
	_ = x[DwarfOpBreg27-139]
	_ = x[DwarfOpBreg28-140]
	_ = x[DwarfOpBreg29-141]
	_ = x[DwarfOpBreg30-142]
	_ = x[DwarfOpBreg31-143]
	_ = x[DwarfOpRegx-144]
	_ = x[DwarfOpFbreg-145]
	_ = x[DwarfOpBregx-146]
	_ = x[DwarfOpPiece-147]
	_ = x[DwarfOpDerefSize-148]
	_ = x[DwarfOpXderefSize-149]
	_ = x[DwarfOpNop-150]
	_ = x[DwarfOpPushObjectAddress-151]
	_ = x[DwarfOpCall2-152]
	_ = x[DwarfOpCall4-153]
	_ = x[DwarfOpCallRef-154]
	_ = x[DwarfOpFormTLSAddress-155]
	_ = x[DwarfOpCallFrameCFA-156]
	_ = x[DwarfOpBitPiece-157]
	_ = x[DwarfOpImplicitValue-158]
	_ = x[DwarfOpStackValue-159]
	_ = x[DwarfOpImplicitPointer-160]
	_ = x[DwarfOpAddrx-161]
	_ = x[DwarfOpConstx-162]
	_ = x[DwarfOpEntryValue-163]
	_ = x[DwarfOpConstType-164]
	_ = x[DwarfOpRegvalType-165]
	_ = x[DwarfOpDerefType-166]
	_ = x[DwarfOpXderefType-167]
	_ = x[DwarfOpConvert-168]
	_ = x[DwarfOpReinterpret-169]
	_ = x[DwarfOpGNUPushTLSAddress-224]
	_ = x[DwarfOpGNUEntryValue-243]
	_ = x[DwarfOpGNUAddrIndex-251]
	_ = x[DwarfOpGNUConstIndex-252]
	_ = x[DwarfOpLLVMFragment-4096]
	_ = x[DwarfOpLLVMConvert-4097]
	_ = x[DwarfOpLLVMTagOffset-4098]
}

const (
	_DwarfOp_name_0 = "DW_OP_addr"
	_DwarfOp_name_1 = "DW_OP_deref"
	_DwarfOp_name_2 = "DW_OP_const1uDW_OP_const1sDW_OP_const2uDW_OP_const2sDW_OP_const4uDW_OP_const4sDW_OP_const8uDW_OP_const8sDW_OP_constuDW_OP_constsDW_OP_dupDW_OP_dropDW_OP_overDW_OP_pickDW_OP_swapDW_OP_rotDW_OP_xderefDW_OP_absDW_OP_andDW_OP_divDW_OP_minusDW_OP_modDW_OP_mulDW_OP_negDW_OP_notDW_OP_orDW_OP_plusDW_OP_plus_uconstDW_OP_shlDW_OP_shrDW_OP_shraDW_OP_xorDW_OP_braDW_OP_eqDW_OP_geDW_OP_gtDW_OP_leDW_OP_ltDW_OP_neDW_OP_skipDW_OP_lit0DW_OP_lit1DW_OP_lit2DW_OP_lit3DW_OP_lit4DW_OP_lit5DW_OP_lit6DW_OP_lit7DW_OP_lit8DW_OP_lit9DW_OP_lit10DW_OP_lit11DW_OP_lit12DW_OP_lit13DW_OP_lit14DW_OP_lit15DW_OP_lit16DW_OP_lit17DW_OP_lit18DW_OP_lit19DW_OP_lit20DW_OP_lit21DW_OP_lit22DW_OP_lit23DW_OP_lit24DW_OP_lit25DW_OP_lit26DW_OP_lit27DW_OP_lit28DW_OP_lit29DW_OP_lit30DW_OP_lit31DW_OP_reg0DW_OP_reg1DW_OP_reg2DW_OP_reg3DW_OP_reg4DW_OP_reg5DW_OP_reg6DW_OP_reg7DW_OP_reg8DW_OP_reg9DW_OP_reg10DW_OP_reg11DW_OP_reg12DW_OP_reg13DW_OP_reg14DW_OP_reg15DW_OP_reg16DW_OP_reg17DW_OP_reg18DW_OP_reg19DW_OP_reg20DW_OP_reg21DW_OP_reg22DW_OP_reg23DW_OP_reg24DW_OP_reg25DW_OP_reg26DW_OP_reg27DW_OP_reg28DW_OP_reg29DW_OP_reg30DW_OP_reg31DW_OP_breg0DW_OP_breg1DW_OP_breg2DW_OP_breg3DW_OP_breg4DW_OP_breg5DW_OP_breg6DW_OP_breg7DW_OP_breg8DW_OP_breg9DW_OP_breg10DW_OP_breg11DW_OP_breg12DW_OP_breg13DW_OP_breg14DW_OP_breg15DW_OP_breg16DW_OP_breg17DW_OP_breg18DW_OP_breg19DW_OP_breg20DW_OP_breg21DW_OP_breg22DW_OP_breg23DW_OP_breg24DW_OP_breg25DW_OP_breg26DW_OP_breg27DW_OP_breg28DW_OP_breg29DW_OP_breg30DW_OP_breg31DW_OP_regxDW_OP_fbregDW_OP_bregxDW_OP_pieceDW_OP_deref_sizeDW_OP_xderef_sizeDW_OP_nopDW_OP_push_object_addressDW_OP_call2DW_OP_call4DW_OP_call_refDW_OP_form_tls_addressDW_OP_call_frame_cfaDW_OP_bit_pieceDW_OP_implicit_valueDW_OP_stack_valueDW_OP_implicit_pointerDW_OP_addrxDW_OP_constxDW_OP_entry_valueDW_OP_const_typeDW_OP_regval_typeDW_OP_deref_typeDW_OP_xderef_typeDW_OP_convertDW_OP_reinterpret"
	_DwarfOp_name_3 = "DW_OP_GNU_push_tls_address"
	_DwarfOp_name_4 = "DW_OP_GNU_entry_value"
	_DwarfOp_name_5 = "DW_OP_GNU_addr_indexDW_OP_GNU_const_index"
	_DwarfOp_name_6 = "DW_OP_LLVM_fragmentDW_OP_LLVM_convertDW_OP_LLVM_tag_offset"
)

var (
	_DwarfOp_index_2 = [...]uint16{0, 13, 26, 39, 52, 65, 78, 91, 104, 116, 128, 137, 147, 157, 167, 177, 186, 198, 207, 216, 225, 236, 245, 254, 263, 272, 280, 290, 307, 316, 325, 335, 344, 353, 361, 369, 377, 385, 393, 401, 411, 421, 431, 441, 451, 461, 471, 481, 491, 501, 511, 522, 533, 544, 555, 566, 577, 588, 599, 610, 621, 632, 643, 654, 665, 676, 687, 698, 709, 720, 731, 742, 753, 763, 773, 783, 793, 803, 813, 823, 833, 843, 853, 864, 875, 886, 897, 908, 919, 930, 941, 952, 963, 974, 985, 996, 1007, 1018, 1029, 1040, 1051, 1062, 1073, 1084, 1095, 1106, 1117, 1128, 1139, 1150, 1161, 1172, 1183, 1194, 1205, 1217, 1229, 1241, 1253, 1265, 1277, 1289, 1301, 1313, 1325, 1337, 1349, 1361, 1373, 1385, 1397, 1409, 1421, 1433, 1445, 1457, 1469, 1479, 1490, 1501, 1512, 1528, 1545, 1554, 1579, 1590, 1601, 1615, 1637, 1657, 1672, 1692, 1709, 1731, 1742, 1754, 1771, 1787, 1804, 1820, 1837, 1850, 1867}
	_DwarfOp_index_5 = [...]uint8{0, 20, 41}
	_DwarfOp_index_6 = [...]uint8{0, 19, 37, 58}
)

func (i DwarfOp) String() string {
	switch {
	case i == 3:
		return _DwarfOp_name_0
	case i == 6:
		return _DwarfOp_name_1
	case 8 <= i && i <= 169:
		i -= 8
		return _DwarfOp_name_2[_DwarfOp_index_2[i]:_DwarfOp_index_2[i+1]]
	case i == 224:
		return _DwarfOp_name_3
	case i == 243:
		return _DwarfOp_name_4
	case 251 <= i && i <= 252:
		i -= 251
		return _DwarfOp_name_5[_DwarfOp_index_5[i]:_DwarfOp_index_5[i+1]]
	case 4096 <= i && i <= 4098:
		i -= 4096
		return _DwarfOp_name_6[_DwarfOp_index_6[i]:_DwarfOp_index_6[i+1]]
	default:
		return "DwarfOp(" + strconv.FormatInt(int64(i), 10) + ")"
	}
}

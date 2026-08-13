"""檢查一條指令的記憶體運算元有沒有被解析成位址(DS 相對存取的診斷)。"""
import sys
import ida_auto, ida_bytes, ida_pro, idautils, idc

def main():
    ida_auto.auto_wait()
    with open(sys.argv[1], "w", encoding="utf-8") as fh:
        for t in sys.argv[2:]:
            ea = int(t, 0)
            fh.write(f"== 0x{ea:X}  {idc.generate_disasm_line(ea,0)}\n")
            for i in range(3):
                ot = idc.get_operand_type(ea, i)
                if ot == 0: break
                fh.write(f"   op{i} type={ot} value=0x{idc.get_operand_value(ea,i):X}\n")
            fh.write(f"   DataRefsFrom = {[hex(x) for x in idautils.DataRefsFrom(ea)]}\n")
        for t in ("0x1093A",):
            ea = int(t, 0)
            fh.write(f"== 目標 0x{ea:X} flags=0x{ida_bytes.get_flags(ea):X} "
                     f"name={idc.get_name(ea)!r} "
                     f"is_data={ida_bytes.is_data(ida_bytes.get_flags(ea))} "
                     f"is_code={ida_bytes.is_code(ida_bytes.get_flags(ea))} "
                     f"word=0x{ida_bytes.get_word(ea):04X}\n")
    ida_pro.qexit(0)

main()

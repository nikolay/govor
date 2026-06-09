import sys

# 6502 opcode table: opcode -> (mnemonic, addressing mode, length)
M = {}
def add(op, mn, mode, ln): M[op]=(mn,mode,ln)
# mode: imp, acc, imm, zp, zpx, zpy, abs, abx, aby, ind, izx, izy, rel
table = """
00 BRK imp 1|01 ORA izx 2|05 ORA zp 2|06 ASL zp 2|08 PHP imp 1|09 ORA imm 2|0A ASL acc 1|0D ORA abs 3|0E ASL abs 3
10 BPL rel 2|11 ORA izy 2|15 ORA zpx 2|16 ASL zpx 2|18 CLC imp 1|19 ORA aby 3|1D ORA abx 3|1E ASL abx 3
20 JSR abs 3|21 AND izx 2|24 BIT zp 2|25 AND zp 2|26 ROL zp 2|28 PLP imp 1|29 AND imm 2|2A ROL acc 1|2C BIT abs 3|2D AND abs 3|2E ROL abs 3
30 BMI rel 2|31 AND izy 2|35 AND zpx 2|36 ROL zpx 2|38 SEC imp 1|39 AND aby 3|3D AND abx 3|3E ROL abx 3
40 RTI imp 1|41 EOR izx 2|45 EOR zp 2|46 LSR zp 2|48 PHA imp 1|49 EOR imm 2|4A LSR acc 1|4C JMP abs 3|4D EOR abs 3|4E LSR abs 3
50 BVC rel 2|51 EOR izy 2|55 EOR zpx 2|56 LSR zpx 2|58 CLI imp 1|59 EOR aby 3|5D EOR abx 3|5E LSR abx 3
60 RTS imp 1|61 ADC izx 2|65 ADC zp 2|66 ROR zp 2|68 PLA imp 1|69 ADC imm 2|6A ROR acc 1|6C JMP ind 3|6D ADC abs 3|6E ROR abs 3
70 BVS rel 2|71 ADC izy 2|75 ADC zpx 2|76 ROR zpx 2|78 SEI imp 1|79 ADC aby 3|7D ADC abx 3|7E ROR abx 3
81 STA izx 2|84 STY zp 2|85 STA zp 2|86 STX zp 2|88 DEY imp 1|8A TXA imp 1|8C STY abs 3|8D STA abs 3|8E STX abs 3
90 BCC rel 2|91 STA izy 2|94 STY zpx 2|95 STA zpx 2|96 STX zpy 2|98 TYA imp 1|99 STA aby 3|9A TXS imp 1|9D STA abx 3
A0 LDY imm 2|A1 LDA izx 2|A2 LDX imm 2|A4 LDY zp 2|A5 LDA zp 2|A6 LDX zp 2|A8 TAY imp 1|A9 LDA imm 2|AA TAX imp 1|AC LDY abs 3|AD LDA abs 3|AE LDX abs 3
B0 BCS rel 2|B1 LDA izy 2|B4 LDY zpx 2|B5 LDA zpx 2|B6 LDX zpy 2|B8 CLV imp 1|B9 LDA aby 3|BA TSX imp 1|BC LDY abx 3|BD LDA abx 3|BE LDX aby 3
C0 CPY imm 2|C1 CMP izx 2|C4 CPY zp 2|C5 CMP zp 2|C6 DEC zp 2|C8 INY imp 1|C9 CMP imm 2|CA DEX imp 1|CC CPY abs 3|CD CMP abs 3|CE DEC abs 3
D0 BNE rel 2|D1 CMP izy 2|D5 CMP zpx 2|D6 DEC zpx 2|D8 CLD imp 1|D9 CMP aby 3|DD CMP abx 3|DE DEC abx 3
E0 CPX imm 2|E1 SBC izx 2|E4 CPX zp 2|E5 SBC zp 2|E6 INC zp 2|E8 INX imp 1|E9 SBC imm 2|EA NOP imp 1|EC CPX abs 3|ED SBC abs 3|EE INC abs 3
F0 BEQ rel 2|F1 SBC izy 2|F5 SBC zpx 2|F6 INC zpx 2|F8 SED imp 1|F9 SBC aby 3|FD SBC abx 3|FE INC abx 3
"""
for entry in table.replace('\n','|').split('|'):
    entry = entry.strip()
    if not entry: continue
    op, mn, mode, ln = entry.split()
    add(int(op,16), mn, mode, int(ln))

def dis(mem, start, end):
    a = start
    lines = []
    while a < end:
        op = mem[a]
        if op not in M:
            lines.append((a, f'{a:04X}: {op:02X}        ???  .byte ${op:02X}'))
            a += 1
            continue
        mn, mode, ln = M[op]
        ops = mem[a+1:a+ln]
        raw = ' '.join(f'{b:02X}' for b in mem[a:a+ln]).ljust(9)
        if mode == 'imp': arg=''
        elif mode == 'acc': arg='A'
        elif mode == 'imm': arg=f'#${ops[0]:02X}'
        elif mode == 'zp': arg=f'${ops[0]:02X}'
        elif mode == 'zpx': arg=f'${ops[0]:02X},X'
        elif mode == 'zpy': arg=f'${ops[0]:02X},Y'
        elif mode == 'abs': arg=f'${ops[0]|(ops[1]<<8):04X}'
        elif mode == 'abx': arg=f'${ops[0]|(ops[1]<<8):04X},X'
        elif mode == 'aby': arg=f'${ops[0]|(ops[1]<<8):04X},Y'
        elif mode == 'ind': arg=f'(${ops[0]|(ops[1]<<8):04X})'
        elif mode == 'izx': arg=f'(${ops[0]:02X},X)'
        elif mode == 'izy': arg=f'(${ops[0]:02X}),Y'
        elif mode == 'rel':
            t = a + 2 + (ops[0] if ops[0]<128 else ops[0]-256)
            arg=f'${t:04X}'
        lines.append((a, f'{a:04X}: {raw} {mn} {arg}'))
        a += ln
    return lines

if __name__ == '__main__':
    mem = open('gowor.mem','rb').read()
    start = int(sys.argv[1], 16)
    end = int(sys.argv[2], 16)
    for _, l in dis(mem, start, end):
        print(l)

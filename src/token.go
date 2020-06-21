package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

const symbols_c = "+-*/;=(),{}<>[]&.!?:|^%~#"

var (
	keywords = map[string]int{
		"_Alignof": TK_ALIGNOF,
		"_Bool":    TK_BOOL,
		"break":    TK_BREAK,
		"case":     TK_CASE,
		"char":     TK_CHAR,
		"continue": TK_CONTINUE,
		"do":       TK_DO,
		"else":     TK_ELSE,
		"extern":   TK_EXTERN,
		"for":      TK_FOR,
		"if":       TK_IF,
		"int":      TK_INT,
		"return":   TK_RETURN,
		"sizeof":   TK_SIZEOF,
		"struct":   TK_STRUCT,
		"switch":   TK_SWITCH,
		"typedef":  TK_TYPEDEF,
		"typeof":   TK_TYPEOF,
		"void":     TK_VOID,
		"while":    TK_WHILE,
	}

	symbols_2 = map[string]int{
		"!=": TK_NE,
		"&&": TK_LOGAND,
		"++": TK_INC,
		"--": TK_DEC,
		"->": TK_ARROW,
		"<<": TK_SHL,
		"<=": TK_LE,
		"==": TK_EQ,
		">=": TK_GE,
		">>": TK_SHR,
		"||": TK_LOGOR,
		"*=": TK_MUL_EQ,
		"/=": TK_DIV_EQ,
		"%=": TK_MOD_EQ,
		"+=": TK_ADD_EQ,
		"-=": TK_SUB_EQ,
		"&=": TK_AND_EQ,
		"^=": TK_XOR_EQ,
		"|=": TK_OR_EQ,
	}

	symbols_3 = map[string]int{
		"<<=": TK_SHL_EQ,
		">>=": TK_SHR_EQ,
	}

	escaped = map[uint8]int{
		'a': '\a',
		'b': '\b',
		'f': '\f',
		'n': '\n',
		'r': '\r',
		't': '\t',
		'v': '\v',
		'e': '\033',
		'E': '\033',
	}
)

type Context struct {
	path   string
	buf    string
	pos    string
	tokens *Vector
	next   *Context
}

type TokenApp struct {
	ctx *Context
	ctx_p *Context_p
}

func read_file(path string) string {
	f := os.Stdin
	if path != "-" {
		f2, err := os.Open(path)
		if err != nil {
			log.Fatal(err)
		}
		f = f2
		defer f2.Close()
	}
	defer f.Close()

	sb := new_sb()
	buf := make([]byte, 4096)
	for {
		n, err := f.Read(buf)
		if n == 0 {
			break
		}
		if err != nil {
			break
		}
		sb_append_n(sb, string(buf[:n]), n)

	}

	if sb.data[sb.len-1] != '\n' {
		sb_add(sb, "\n")
	}
	return sb_get(sb)
}

func new_ctx(next *Context, path, buf string) *Context {
	ctx := new(Context)
	ctx.path = path
	ctx.buf = buf
	ctx.pos = ctx.buf
	ctx.tokens = new_vec()
	ctx.next = next
	return ctx
}

// Error reporting

// Finds a line pointed by a given pointer from the input line
// to print it out.
func print_line(buf string, path string, pos int) {
	curline, s := buf, buf
	line, col := 0, 0
	pos_ := buf[pos:]

	for i, c := range buf {

		if c == '\n' {
			curline = buf[i+1:]
			line++
			col = 0
			s = buf[i+1:]
			continue
		}

		if s != pos_ {
			col++
			s = buf[i+1:]
			continue
		}

		fmt.Fprintf(os.Stderr, "errorReport at %s:%d:%d\n\n", path, line+1, col+1)
		for i, c2 := range curline {
			if c2 == '\n' {
				curline = curline[:i]
				break
			}
		}
		fmt.Fprintf(os.Stderr, "%s\n", curline)

		for i := 0; i < col-1; i++ {
			fmt.Fprintf(os.Stderr, " ")
		}
		fmt.Fprintf(os.Stderr, "^\n\n")
		return
	}
}

func bad_token(t *Token, msg string) {
	print_line(t.ctx.buf, t.ctx.path, t.start)
	errorReport(msg)
}

func tokstr(t *Token) string {
	// assert(t.start && t.end)
	buf := t.ctx.buf
	return strndup(buf[t.start:], t.end-t.start)
}

func line(t *Token) int {
	buf := t.ctx.buf
	ll := len(buf)
	n := 1
	for i := 0; i < ll-t.end; i++ {
		if buf[i] == '\n' {
			n++
		}
	}
	return n
}

// Atomic unit in the grammer is called "token".
// For example, `123`, `"abc"` and `while` are tokens.
// The tokenizer splits an inpuit string into tokens.
// Spaces and comments are removed by the tokenizer.

func (ctx *Context) add_t(ty, start int) *Token {
	t := new(Token)
	t.ty = ty
	t.start = start
	t.ctx = ctx
	vec_push(ctx.tokens, t)
	return t
}

func (ctx *Context) block_comment(idx int) int {
	buf := ctx.buf
	ll := len(buf)
	for ; idx < ll; idx++ {
		if startswith("*/", idx, buf) {
			return idx + 2
		}
	}

	errorReport("unclosed comment")
	return -1
}

func (ctx *Context) c_char(val *int, idx int) int {
	buf := ctx.buf
	char := buf[idx]
	if char != '\\' {
		*val = int(char)
		return idx + 1
	}

	idx += 1
	char = buf[idx]
	esc, ok := escaped[char]
	if ok {
		*val = esc
		return idx + 1
	}
	if char == 'x' {
		tmp := 0
		idx += 1
		char = buf[idx]
		for isxdigit_char(char) {
			tmp = tmp*16 + isxdigit_val(char)
			idx += 1
			char = buf[idx]
		}
		*val = tmp
		return idx
	}

	if isoctal_char(char) {
		tmp := isoctal_val(char)
		idx += 1
		char = buf[idx]
		for isoctal_char(char) {
			tmp = tmp*8 + isoctal_val(char)
			idx += 1
			char = buf[idx]
		}
		*val = tmp
		return idx
	}

	*val = int(char)
	return idx + 1
}

func (ctx *Context) char_literal(idx int) int {
	buf := ctx.buf
	idx += 1

	t := ctx.add_t(TK_NUM, idx)
	idx = ctx.c_char(&t.val, idx)

	char := buf[idx]
	if char != '\'' {
		errorReport("unclosed character literal")
	}
	idx += 1
	t.end = idx
	return idx
}

func (ctx *Context) string_literal(idx int) int {
	buf := ctx.buf
	ll := len(buf)

	idx += 1
	t := ctx.add_t(TK_STR, idx)
	sb := new_sb()
	char := buf[idx]
	for char != '"' {
		tmp := 0
		idx = ctx.c_char(&tmp, idx)
		sb_add(sb, string(tmp))

		if idx >= ll {
			bad_token(t, "unclosed string literal")
		}
		char = buf[idx]
		if char == '\n' {
			bad_token(t, "newline in string literal")
		}
	}
	t.str = sb_get(sb)
	t.len = len(t.str)
	idx += 1
	t.end = idx
	return idx
}

func (ctx *Context) ident_t(idx int) int {
	buf := ctx.buf
	ilen := 1
	char := buf[ilen]
	for isalpha_char(char) || isdigit_char(char) || char == '_' {
		ilen++
		char = buf[ilen]
	}

	name := buf[idx : idx+ilen]
	ty, ok := keywords[name]
	if !ok {
		ty = TK_IDENT
	}

	t := ctx.add_t(ty, idx)
	t.name = name
	idx += ilen
	t.end = idx
	return idx
}

func (ctx *Context) hexadecimal(idx int) int {
	buf := ctx.buf
	t := ctx.add_t(TK_NUM, idx)

	idx += 2
	char := buf[idx]
	if !isxdigit_char(char) {
		bad_token(t, "bad hexadecimal number")
	}

	tmp := 0
	for isxdigit_char(char) {
		tmp = tmp*16 + isxdigit_val(char)
		idx += 1
		char = buf[idx]
	}

	t.val = tmp
	t.end = idx
	return idx
}

func (ctx *Context) octal(idx int) int {
	buf := ctx.buf
	t := ctx.add_t(TK_NUM, idx)

	char := buf[idx]
	tmp := 0
	for isoctal_char(char) {
		tmp = tmp*8 + isoctal_val(char)

		idx += 1
		char = buf[idx]
	}

	t.val = tmp
	t.end = idx
	return idx
}

func (ctx *Context) decimal(idx int) int {
	buf := ctx.buf
	t := ctx.add_t(TK_NUM, idx)

	char := buf[idx]
	tmp := 0
	for isdigit_char(char){
		tmp = tmp * 10 + isdigit_val(char)

		idx += 1
		char = buf[idx]
	}

	t.val = tmp
	t.end = idx
	return idx
}

func (ctx *Context) number(idx int) int {
	buf := ctx.buf
	if startswith("0x", idx, buf) || startswith("0X", idx, buf) {
		return ctx.hexadecimal(idx)
	}

	char := buf[idx]
	if char == '0' {
		return ctx.octal(idx)
	}

	return ctx.decimal(idx)
}

// Tokenized input is stored to this array
func (ctx *Context) scan() {
	buf := ctx.buf
	idx := 0
	ll := len(buf)
	for idx < ll {
		char := buf[idx]
		if char == '\n' {
			t := ctx.add_t(int(char), idx)
			idx += 1
			t.end = idx
			continue
		}

		if char == ' ' || char == '\t' || char == '\v' || char == '\f' {
			idx += 1
			continue
		}

		if startswith("//", idx, buf) {
			for idx < ll && buf[idx] != '\n' {
				idx += 1
			}
			continue
		}

		if startswith("/*", idx, buf) {
			idx = ctx.block_comment(idx)
			continue
		}

		if char == '\'' {
			idx = ctx.char_literal(idx)
			continue
		}

		if char == '"' {
			idx = ctx.string_literal(idx)
			continue
		}

		if idx+1 < ll {
			symbol := buf[idx : idx+2]
			ty, ok := symbols_2[symbol]
			if ok {
				t := ctx.add_t(ty, idx)
				idx += len(symbol)
				t.end = idx
				continue
			}
		}

		if idx+2 < ll {
			symbol := buf[idx : idx+3]
			ty, ok := symbols_3[symbol]
			if ok {
				t := ctx.add_t(ty, idx)
				idx += len(symbol)
				t.end = idx
				continue
			}
		}

		if strchr("+-*/;=(),{}<>[]&.!?:|^%~#", char) >= 0 {
			t := ctx.add_t(int(char), idx)
			idx += 1
			t.end = idx
			continue
		}

		if isalpha_char(char) || char == '_' {
			idx = ctx.ident_t(idx)
			continue
		}

		if isdigit_char(char) {
			idx = ctx.number(idx)
			continue
		}

		print_line(ctx.buf, ctx.path, idx)
		errorReport("cannot tokenize")
	}
}

func canonicalize_newline(p string) string {
	return strings.Replace(p, "\r\n", "\n", -1)
}

func remove_backslash_newline(p string) string {
	return strings.Replace(p, "\\\n", "", -1)
}

func remove_pragma_newline(p string) string {
	p = strings.Replace(p, "\n#pragma", "\n// #pragma", -1)
	p = strings.Replace(p, "\n__pragma", "\n// __pragma", -1)
	return p
}

func strip_newline_tokens(tokens *Vector) *Vector {
	v := new_vec()
	for i := 0; i < tokens.len; i++ {
		t := tokens.data[i].(*Token)
		if t.ty != '\n' {
			vec_push(v, t)
		}
	}
	return v
}

func append_t(x, y *Token) {
	sb := new_sb()
	sb_append_n(sb, x.str, x.len)
	sb_append_n(sb, y.str, y.len)
	x.str = sb_get(sb)
	x.len = sb.len
}

func join_string_literals(tokens *Vector) *Vector {
	v := new_vec()
	var last *Token

	for i := 0; i < tokens.len; i++ {
		t := tokens.data[i].(*Token)
		if last != nil && last.ty == TK_STR && t.ty == TK_STR {
			append_t(last, t)
			continue
		}

		last = t
		vec_push(v, t)
	}
	return v
}

func tokenize(path string, add_eof bool, ctx *Context) *Vector {
	buf := read_file(path)
	buf = canonicalize_newline(buf)
	buf = remove_backslash_newline(buf)
	buf = remove_pragma_newline(buf)
	app := &TokenApp{
		ctx: new_ctx(ctx, path, buf),
	}
	app.ctx.scan()
	if add_eof {
		app.ctx.add_t(TK_EOF, -1)
	}

	v := app.ctx.tokens
	app.ctx = app.ctx.next

	v = app.preprocess(v)
	v = strip_newline_tokens(v)
	return join_string_literals(v)
}

// debug
func print_tokens(tokens *Vector) {
	m := map[int]string{
		TK_NUM:     "TK_NUM      ",
		TK_STR:     "TK_STR      ",
		TK_IDENT:   "TK_IDENT    ",
		TK_ARROW:   "TK_ARROW    ",
		TK_EXTERN:  "TK_EXTERN   ",
		TK_TYPEDEF: "TK_TYPEDEF  ",
		TK_INT:     "TK_INT      ",
		TK_CHAR:    "TK_CHAR     ",
		TK_VOID:    "TK_VOID     ",
		TK_STRUCT:  "TK_STRUCT   ",
		TK_IF:      "TK_IF       ",
		TK_ELSE:    "TK_ELSE     ",
		TK_FOR:     "TK_FOR      ",
		TK_DO:      "TK_DO       ",
		TK_WHILE:   "TK_WHILE    ",
		TK_BREAK:   "TK_BREAK    ",
		TK_EQ:      "TK_EQ       ",
		TK_NE:      "TK_NE       ",
		TK_LE:      "TK_LE       ",
		TK_GE:      "TK_GE       ",
		TK_LOGOR:   "TK_LOGOR    ",
		TK_LOGAND:  "TK_LOGAND   ",
		TK_SHL:     "TK_SHL      ",
		TK_SHR:     "TK_SHR      ",
		TK_INC:     "TK_INC      ",
		TK_DEC:     "TK_DEC      ",
		TK_MUL_EQ:  "TK_MUL_EQ   ",
		TK_DIV_EQ:  "TK_DIV_EQ   ",
		TK_MOD_EQ:  "TK_MOD_EQ   ",
		TK_ADD_EQ:  "TK_ADD_EQ   ",
		TK_SUB_EQ:  "TK_SUB_EQ   ",
		TK_SHL_EQ:  "TK_SHL_EQ   ",
		TK_SHR_EQ:  "TK_SHR_EQ   ",
		TK_AND_EQ:  "TK_BITAND_EQ",
		TK_XOR_EQ:  "TK_XOR_EQ   ",
		TK_OR_EQ:   "TK_BITOR_EQ ",
		TK_RETURN:  "TK_RETURN   ",
		TK_SIZEOF:  "TK_SIZEOF   ",
		TK_ALIGNOF: "TK_ALIGNOF  ",
		TK_TYPEOF:  "TK_TYPEOF  ",
		TK_PARAM:   "TK_PARAM    ",
		TK_EOF:     "TK_EOF      ",
	}
	for i := 0; i < tokens.len; i++ {
		t := tokens.data[i].(*Token)
		s, ok := m[t.ty]
		if !ok {
			if t.ty != '\n' {
				s = fmt.Sprintf("%c           ", t.ty)
			} else {
				s = "[LF]         "
			}
		}
		val := ""
		if t.ty == TK_NUM {
			val = strconv.Itoa(t.val)
		} else {
			val = t.name
		}
		fmt.Printf("[%03d] %s %s\n", i+1, s, val)
	}
	fmt.Println()
}

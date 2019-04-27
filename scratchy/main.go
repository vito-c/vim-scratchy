package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/neovim/go-client/nvim"
	"github.com/neovim/go-client/nvim/plugin"
	"github.com/vito-c/scratchy/jq"
)

type Svim struct {
	*nvim.Nvim
}

type SBuffer struct {
	nvim.Buffer
	nvim *Svim
	data [][]byte
}

// Buffer Options - New Location
const (
	DEF  = "enew"
	HORZ = "new"
	TAB  = "tabnew"
	VERT = "vnew"
)

// Buffer Options - Type
const (
	normal   = "normal"   // normal buffer
	acwrite  = "acwrite"  // buffer will always be written with |BufWriteCmd|s
	help     = "help"     // help buffer (do not set this manually)
	nofile   = "nofile"   // buffer is not related to a file, will not be written
	nowrite  = "nowrite"  // buffer will not be written
	quickfix = "quickfix" // list of errors |:cwindow| or locations |:lwindow|
	terminal = "terminal" // |terminal-emulator| buffer
)

const (
	hide   = "hide"
	unload = "unload"
	delete = "delete"
	wipe   = "wipe"
)

type BufferOptions struct {
	Number     bool
	Buflisted  bool
	Bufhidden  string
	Location   string
	Modifiable bool
	Modified   bool
	Name       string
	ReadOnly   bool
	BufferType string
}

type jqRun struct {
	nvim       *nvim.Nvim
	forceSetup bool
	showError  bool
	args       []string
}

func (v *Svim) NewBufferOpts(options BufferOptions) SBuffer {
	v.Command(options.Location) // tab, vert, horz, default
	buff, _ := v.CurrentBuffer()
	log.Println("new buff: ", buff, " name: ", options.Name)
	v.SetBufferName(buff, options.Name)
	v.SetBufferOption(buff, "buftype", options.BufferType)
	return SBuffer{buff, v, [][]byte{}}
}

func (v *Svim) NewVSplitBuffer(name string) SBuffer {
	return v.NewBufferOpts(BufferOptions{
		Location:   VERT,
		Modifiable: true,
		Modified:   false,
		ReadOnly:   false,
		Name:       name,
		BufferType: normal,
	})
}

func (v *Svim) NewTabBuffer(name string) SBuffer {
	return v.NewBufferOpts(BufferOptions{
		Location:   TAB,
		Modifiable: true,
		Modified:   false,
		ReadOnly:   false,
		Name:       name,
		BufferType: normal,
	})
}

func (v *Svim) NewBuffer(name string) SBuffer {
	return v.NewBufferOpts(BufferOptions{
		Location:   DEF,
		Modifiable: true,
		Modified:   false,
		ReadOnly:   false,
		Name:       name,
		BufferType: normal,
	})
}

func (v *Svim) CreateSBuffer(buff nvim.Buffer, err error) SBuffer {
	log.Println("BUFFER NUMBEr: ", buff)
	data := [][]byte{}
	return SBuffer{
		Buffer: buff,
		nvim:   v,
		data:   data,
	}
}

func (b *SBuffer) readonly(isReadOnly bool) {
	log.Println("REAADONLY ", b.Buffer, " ", isReadOnly)
	// b.nvim.Nvim.SetBufferOption(b.Buffer, "modifiable", isReadOnly)
	// b.nvim.Nvim.SetBufferOption(b.Buffer, "modified", isReadOnly)
	// b.nvim.Nvim.SetBufferOption(b.Buffer, "readonly", isReadOnly)
	b.nvim.Nvim.SetCurrentBuffer(b.Buffer)
	if isReadOnly {
		b.nvim.Nvim.Command("setlocal noma nomod nonu ro nornu")
	} else {
		b.nvim.Nvim.Command("setlocal ma mod nu ro nornu")
	}
}

var databuff, jqbuff, outbuff SBuffer
var dataWin, jqWin, outWin nvim.Window

func setup(n *nvim.Nvim, args []string) (string, error) { // Declare first arg as *nvim.Nvim to get current client
	log.Println("IN SETUP")
	jq.Init()
	jq.Path = "/usr/local/bin/jq"
	v := Svim{n}
	databuff = v.CreateSBuffer(n.CurrentBuffer())
	dataWin, _ = v.CurrentWindow()
	outbuff = v.NewTabBuffer("[Output]")
	v.Command("set syntax=json")
	outWin, _ = v.CurrentWindow()
	log.Println("outWin ", outWin)

	/*** Filter Buffer ***/
	jqbuff = v.NewBufferOpts(BufferOptions{
		Location:   VERT,
		Modifiable: true,
		Modified:   false,
		ReadOnly:   false,
		Name:       "[jq]",
		BufferType: nofile,
	})
	log.Println("jqbuff: ", jqbuff.Buffer)
	v.Command("set syntax=jq")

	// Finish setup
	v.SetCurrentBuffer(databuff.Buffer)
	dataWin, _ = v.CurrentWindow()
	log.Println("dataWin ", dataWin)
	v.Command("split")
	v.Command("wincmd K")
	jqWin, _ = v.CurrentWindow()
	v.SetWindowHeight(jqWin, 10)
	log.Println("dataWin ", jqWin)

	v.SetCurrentBuffer(jqbuff.Buffer)
	v.SetCurrentLine([]byte("map(.url)"))
	log.Printf("jqbuff %v\n", jqbuff)
	log.Printf("databuff %v\n", databuff)
	log.Printf("outbuff %v\n", outbuff)
	v.SetBufferAuCmd("TextChangedI", jqbuff.Buffer, "call ScratchyRun()")
	v.SetBufferAuCmd("TextChanged", jqbuff.Buffer, "call ScratchyRun()")
	return scratchyRun(n, false, true, args)
}

func (r *SBuffer) getString() string {
	r.LoadData()
	var buff []byte

	for _, b := range r.data {
		buff = append(buff, b[:]...)
	}
	return string(buff)
}

var jqr *jq.JQ

func scratchyRun(
	n *nvim.Nvim,
	forceSetup bool,
	showError bool,
	args []string,
) (string, error) { // Declare first arg as *nvim.Nvim to get current client
	v := Svim{n}
	if (jqbuff.Buffer == databuff.Buffer) || forceSetup {
		return setup(v.Nvim, args)
	}

	json := databuff.getString()
	query := jqbuff.getString()

	jqr = &jq.JQ{
		J: string(json),
		Q: string(query),
	}

	v.SetCurrentWindow(outWin)
	v.SetCurrentBuffer(outbuff.Buffer)
	v.Command("setlocal bt=nofile bh=wipe ma mod nonu nobl nowrap noro nornu")
	log.Println("-------------------")
	if err := jqr.Validate(); err != nil {
		log.Println("ERR: ", err.Error())
		return err.Error(), nil
	}
	log.Println("-------------------")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	log.Println(jqr.String())
	if err := jqr.Eval(ctx, &outbuff, ioutil.Discard); err != nil {
		if showError {
			v.Command("echom \"" + err.Error() + "\"")
		}
		log.Println(err.Error())
	}
	v.Command("setlocal bt=nofile bh=wipe noma nomod nonu nobl nowrap ro nornu")
	v.SetCurrentWindow(jqWin)

	return "finished", nil
}

// func JqTextChanged(jqr jqRun) (string, error) { // Declare first arg as *nvim.Nvim to get current client
// 	scratchyRun(jqr.nvim, jqr.forceSetup, jqr.showError, jqr.
// }
//
// func JqTextChangedI() {
// }

func (v *Svim) RemoveBufferAuCmds() {
	return
}

func (v *Svim) SetBufferAuCmd(aucmd string, buffer nvim.Buffer, cmd string) {
	str := fmt.Sprintf("au %s <buffer=%d> %s", aucmd, buffer, cmd)
	log.Println("cmd: ", str)
	if err := v.Command(str); err != nil {
		log.Fatalln(err)
	}
}

func (r *SBuffer) eof() error {
	if len(r.data) == 0 {
		return io.EOF
	}
	if len(r.data) == 1 && len(r.data[0]) == 0 {
		return io.EOF
	}
	return nil
}

func (r *SBuffer) readByte() (b byte, err error) {
	if r.eof() != nil {
		return 0, io.EOF
	}
	b = r.data[0][0]
	r.data[0] = r.data[0][1:]
	return b, nil
}

func (b *SBuffer) LoadData() (err error) {
	b.data, err = b.nvim.Nvim.BufferLines(b.Buffer, 0, -1, true)
	return err
}

func (b *SBuffer) Read(p []byte) (n int, err error) {
	if b.eof() != nil {
		return 0, io.EOF
	}

	if c := cap(p); c > 0 {
		for n < c {
			p[n], _ = b.readByte()
			n++
			if b.eof() != nil {
				break
			}
		}
	}
	return 0, nil
}

func (b *SBuffer) Write(p []byte) (n int, err error) {
	log.Println("Do Writer: ", string(p))

	if len(p) == 0 {
		return 0, nil
	}

	err = b.nvim.SetBufferLines(
		b.Buffer,
		0, -1, true, bytes.Split(p, []byte{'\n'}))
	if err != nil {
		log.Println("write error: ", err.Error())
	}
	return len(p), err
}

// func (b *SBuffer) Write(p []byte) (n int, err error) {
// 	return 0, nil
// }

func main() {
	logPath := "/Users/vitocutten/.local/share/nvim/scratchy.log"
	if os.ExpandEnv("${NVIM_SCRATCHY_LOG_FILE}") != "" {
		logPath = os.ExpandEnv("${NVIM_SCRATCHY_LOG_FILE}")
	}

	f, _ := os.OpenFile(
		logPath,
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0666,
	)
	defer f.Close()
	log.SetOutput(f)
	plugin.Main(func(p *plugin.Plugin) error {
		p.HandleFunction(&plugin.FunctionOptions{Name: "MyTest"}, setup)
		p.HandleFunction(&plugin.FunctionOptions{Name: "ScratchyRun"}, scratchyRun)
		return nil
	})
}

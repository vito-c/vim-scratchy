package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/jingweno/jqplay/jq"
	"github.com/neovim/go-client/nvim"
	"github.com/neovim/go-client/nvim/plugin"
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

type BufferOptions struct {
	Location   string
	Modifiable bool
	Modified   bool
	Name       string
	ReadOnly   bool
	BufferType string
}

func (v *Svim) NewBufferOpts(options BufferOptions) SBuffer {
	v.Command(options.Location) // tab, vert, horz, default
	buff, _ := v.CurrentBuffer()
	v.SetBufferName(buff, options.Name)
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

func readonly(v *nvim.Nvim, buff nvim.Buffer, readable bool) {
	v.SetBufferOption(buff, "modifiable", readable)
	v.SetBufferOption(buff, "modified", readable)
	v.SetBufferOption(buff, "readonly", readable)
}

var databuff, jqbuff, outbuff nvim.Buffer

func setup(n *nvim.Nvim, args []string) (string, error) { // Declare first arg as *nvim.Nvim to get current client
	log.Println("IN SETUP")
	// var ft string
	v := Svim{n}
	databuff, _ = v.CurrentBuffer()
	/*** OUTPUT Buffer ***/
	v.Command("tabnew")
	outbuff, _ = v.CurrentBuffer()
	v.SetBufferName(outbuff, "[Output]")
	v.Command("set syntax=json")
	v.Command("setlocal bt=nofile bh=wipe noma nomod nonu nobl nowrap ro nornu")

	/*** Filter Buffer ***/
	// jqbuff := v.NewBufferOpts(BufferOptions{
	// 	Location:   VERT,
	// 	Modifiable: true,
	// 	Modified:   false,
	// 	ReadOnly:   false,
	// 	Name:       "[jq]",
	// 	BufferType: nofile,
	// })
	v.Command("vnew")
	jqbuff, _ = v.CurrentBuffer()
	v.SetBufferOption(jqbuff, "buftype", "nofile")
	v.Command("set syntax=jq")
	v.SetBufferName(jqbuff, "[jq]")

	// Finish setup
	v.SetCurrentBuffer(databuff)
	v.Command("split")
	v.Command("wincmd K")
	cw, _ := v.CurrentWindow()
	v.SetWindowHeight(cw, 10)
	v.SetCurrentBuffer(jqbuff)
	// v.SetBufferAuCmd("TextChangedI", jqbuff, "ScratchyRun")
	// v.SetBufferAuCmd("TextChanged", jqbuff, "ScratchyRun")
	v.SetCurrentLine([]byte(".[]|.url"))
	log.Printf("jqbuff %v\n", jqbuff)
	log.Printf("databuff %v\n", databuff)
	return scratchyRun(n, args)

	// jq := &JQ{
	// 			J: `{ "foo": { "bar": { "baz": 123 } } }`,
	// 			Q: ".",
	// 		}
	// 		err := jq.Eval(ctx, ioutil.Discard)
	// 		if err != nil {
	// 			t.Errorf("err should be nil: %s", err)
	// 		}
	// return "Test: " + ft, nil
}

func scratchyRun(n *nvim.Nvim, args []string) (string, error) { // Declare first arg as *nvim.Nvim to get current client
	if jqbuff == databuff {
		return setup(n, args)
	}
	v := Svim{n}
	qb, err := v.BufferLines(jqbuff, 0, -1, true)
	if err != nil {
		log.Println("Error in jqbuff")
		log.Fatal(err)
	}
	jb, err := v.BufferLines(databuff, 0, -1, true)
	if err != nil {
		log.Println("Error in databuff")
		log.Fatal(err)
	}
	jqbuff.String()

	query := []byte{}
	json := []byte{}
	for _, b := range qb {
		query = append(query, b[:]...)
	}
	// qstr := string(query)
	// log.Println("================")
	// log.Println(qstr)
	// log.Println("====================")

	for _, b := range jb {
		log.Println(string(b[:]))
		json = append(json, b[:]...)
	}
	// jstr := string(json)
	// log.Println("====================")
	// log.Println(jstr)
	// log.Println("====================")

	jqr := &jq.JQ{
		J: string(json),
		Q: string(query),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	// if err := jqr.Eval(ctx, ); err == nil {
	// }

	if err != nil {
		log.Fatalln(err)
		// t.Errorf("err should be nil: %s", err)
	}
	return "finished", nil
}

func (v *Svim) SetBufferAuCmd(aucmd string, buffer nvim.Buffer, cmd string) {
	v.Command(fmt.Sprintf("au %v <buffer=%v> %v", aucmd, buffer, cmd))
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
	return 0, nil
}

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

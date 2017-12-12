" Header {{{
"--------------------------------------------------------------------------------
" vim: set sw=4 ts=4 sts=4 et tw=90  foldmethod=marker :
"
" By: Vito C.
" }}}
let s:was_bash = 0
function! scratchy#scala(...) abort " {{{
    tabnew
endfunction " }}}

function! scratchy#bashClear(...) abort " {{{
    let bname = bufname('')
    let bnum = bufnr(bname)
    echo 'bnum: ' . bnum
    if bname !=# '[Output]'
        buffer \[Output]
    endif
    setlocal ma noro mod
    normal! ggdG
    setlocal noma ro nomod
    if bname !=# '[Output]'
        exec 'buffer ' . bnum
    else
        wincmd w
    endif
endfunction " }}}

function! scratchy#bashClose(...) abort " {{{
    bw \[Bash]
    tabclose
endfunction " }}}

" function! scratchy#bash(...) abort " {{{
"     tabnew
"     setlocal bt=nofile bh=wipe noma nomod nonu nobl nowrap ro nornu
"     silent file [Output]
"     vnew
"     let s:was_bash = g:is_bash
"     let g:is_bash = 1
"     set syntax=sh
"     set filetype=sh
"     setlocal bt=nofile
"     silent file \[Bash]
"     let g:is_bash = s:was_bash
" endfunction " }}}

let s:bashbacks = {
            \ 'on_stdout': function('scratchy#bashandler'),
            \ 'on_stderr': function('scratchy#bashandler'),
            \ 'on_exit': function('scratchy#bashandler')
            \ }

function! scratchy#bashandler(job_id, data, event) abort
    "let str = 'shell: ' . self.shell . ' job_id: ' . a:job_id . ' data: ' . a:data . ' event: ' . a:event
    let str = 'shell: ' . self.shell . ' event: ' . a:event . ' job_id: ' . a:job_id
    if a:event !=# 'exit'
        let str = str . ' data: ' . join(a:data)
    endif
    echo str
    if a:event ==# 'stdout'
        buffer \[Output]
        setlocal ma noro mod
        "call append('$', a:data)
        for line in a:data
            if line ==? ''
                continue
            endif
            call append(line('$'), line)
        endfor
        setlocal noma ro nomod
        buffer \[Bash]
    elseif a:event ==# 'stderr'
        buffer \[Output]
        setlocal ma noro mod
        for line in a:data
            call append(line('$'), 'Error ' . line)
        endfor
        setlocal noma ro nomod
        buffer \[Bash]
    endif
endfunction

" function! scratchy#bashrun() " {{{
"     buffer \[Bash]
"     let scmd = join(getline(1,'$'))
"     echo scmd
"     let bashjob = jobstart(['bash', '-c', scmd],
"                 \ extend({'shell': 'scratchbash' }, s:bashbacks))
" endfunction " }}}

" ------------------------------------------------------------------
" Function Callers
" ------------------------------------------------------------------
function! s:w(bang)
  return a:bang ? {} : copy(get(g:, 'fzf_layout', g:fzf#vim#default_layout))
endfunction


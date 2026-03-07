package completions

const BashCompletion = `
_stx_completion() {
    local cur prev words cword
    _init_completion || return

    local commands="publish nickname articles subjects subject file dumpdb set-credentials"

    if [[ ${cword} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "${commands}" -- "$cur") )
        return
    fi

    case "${words[1]}" in

        nickname)
            COMPREPLY=( $(compgen -W "create import import-content edit remove rename list" -- "$cur") )
            ;;

        subject)
            COMPREPLY=( $(compgen -W "add delete rename" -- "$cur") )
            ;;

        file)
            COMPREPLY=( $(compgen -W "upload delete list" -- "$cur") )
            ;;

    esac
}

complete -F _stx_completion stx
`

const ZshCompletion = `
#compdef stx

_stx() {
    local -a commands
    commands=(
        "set-credentials:Set API credentials"
        "publish:Publish an article"
        "nickname:Manage nicknames"
        "articles:List articles"
        "subjects:List subjects"
        "subject:Manage subjects"
        "file:Manage files"
        "dumpdb:Download database dump"
        "completion:Generate shell completion"
    )

    if (( CURRENT == 2 )); then
        _describe 'command' commands
        return
    fi

    case "$words[2]" in

        nickname)
            local -a subcmds
            subcmds=(
                "create:Create nickname"
                "import:Import nickname"
                "import-content:Import article content"
                "edit:Edit nickname metadata"
                "remove:Remove nickname"
                "rename:Rename nickname"
                "list:List nicknames"
            )

            if (( CURRENT == 3 )); then
                _describe 'nickname command' subcmds
            fi
            ;;

        subject)
            local -a subcmds
            subcmds=(
                "add:Add subject"
                "delete:Delete subject"
                "rename:Rename subject"
            )

            if (( CURRENT == 3 )); then
                _describe 'subject command' subcmds
            fi
            ;;

        file)
            local -a subcmds
            subcmds=(
                "upload:Upload file"
                "delete:Delete file"
                "list:List files"
            )

            if (( CURRENT == 3 )); then
                _describe 'file command' subcmds
            fi
            ;;

    esac
}

_stx "$@"
`




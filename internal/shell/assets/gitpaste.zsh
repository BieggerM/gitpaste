# gitpaste Zsh command-not-found integration.
# This file is sourced by ~/.zshrc; it is not intended to be executed.

if (( ${+_gitpaste_zsh_hook_loaded} )); then
    return 0 2>/dev/null || exit 0
fi
typeset -g _gitpaste_zsh_hook_loaded=1

if (( $+functions[command_not_found_handler] )); then
    functions[_gitpaste_previous_command_not_found_handler]=${functions[command_not_found_handler]}
fi

_gitpaste_zsh_url_candidate() {
    [[ $1 =~ ^https://(github\.com|gitlab\.com)/[A-Za-z0-9._-]+(/[A-Za-z0-9._-]+)+$ ]] ||
        [[ $1 =~ ^git@(github\.com|gitlab\.com):[A-Za-z0-9._-]+(/[A-Za-z0-9._-]+)+$ ]]
}

command_not_found_handler() {
    if (( $# == 1 )) && _gitpaste_zsh_url_candidate "$1"; then
        command gitpaste clone -- "$1"
        return $?
    fi

    if (( $+functions[_gitpaste_previous_command_not_found_handler] )); then
        _gitpaste_previous_command_not_found_handler "$@"
        return $?
    fi

    printf 'zsh: command not found: %s\n' "$1" >&2
    return 127
}

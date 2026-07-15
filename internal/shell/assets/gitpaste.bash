# gitpaste Bash command-not-found integration.
# This file is sourced by ~/.bashrc; it is not intended to be executed.

if declare -F command_not_found_handle >/dev/null 2>&1; then
    printf '%s\n' 'gitpaste: Bash command_not_found_handle already exists; hook not installed (safe chaining is unavailable without dynamic evaluation)' >&2
    return 0 2>/dev/null || exit 0
fi

_gitpaste_bash_url_candidate() {
    [[ $1 =~ ^https://(github\.com|gitlab\.com)/[A-Za-z0-9._-]+(/[A-Za-z0-9._-]+)+$ ]] ||
        [[ $1 =~ ^git@(github\.com|gitlab\.com):[A-Za-z0-9._-]+(/[A-Za-z0-9._-]+)+$ ]]
}

command_not_found_handle() {
    if (( $# == 1 )) && _gitpaste_bash_url_candidate "$1"; then
        command gitpaste clone -- "$1"
        return $?
    fi

    printf '%s: %s: command not found\n' "${0##*/}" "$1" >&2
    return 127
}

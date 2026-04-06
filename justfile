
_git_is_dirty:
        @git diff-index --quiet --cached HEAD --

_old_tag := `git describe --tags --abbrev=0`
_next_minor := `git describe --tags --abbrev=0 | cut -d . -f 2 | awk '{print $1 + 1}'`
_next_tag := f"v0.{{ _next_minor }}.0"

release-next: _git_is_dirty
    sed -i 's/{{_old_tag}}/{{_next_tag}}/g' main.go flake.nix
    git add main.go flake.nix
    git commit -m "{{ _next_tag }}"
    git tag -a {{ _next_tag }} -m "new version"
    git push
    git push origin {{ _next_tag }}



package files

// jobFuncs start the scripts of jobs. SIGTERM (Cancel) and a lost stream (HUP, or PIPE on the next
// output) end a script, so its EXIT traps remove what is left half done. The functions print what
// progressWriter reads:
//   - pack OUT EXCLUDE ENTRY… packs entries ("./name") of the current folder, without the folder
//     EXCLUDE, into the tar.gz OUT under a temporary name. It prints "total N" and every entry
//     (tar writes the names unbuffered only while the archive goes to stdout).
//   - unpack ARCHIVE DEST extracts a tar or zip archive. A tar reports the bytes read, a zip the
//     files it unpacks.
//   - report_reads PID SIZE prints "pos N SIZE" every second while PID runs (N: read position of fd 3).
//   - lines and tar_names keep the "pos" lines, tar's entries and its errors apart.
const jobFuncs = `trap 'exit 143' TERM INT HUP PIPE
set -o pipefail
lines() { while IFS= read -r l; do echo "$l"; done; }
tar_names() {
  while IFS= read -r l; do
    case $l in ./*) echo "$l" ;; *) echo "$l" >&2 ;; esac
  done
}
report_reads() {
  while kill -0 "$1" 2>/dev/null; do
    echo "pos $(sed -n 's/^pos:[[:space:]]*//p' /proc/$$/fdinfo/3 2>/dev/null) $2"
    sleep 1
  done
  wait "$1"
}
pack() {
  local out="$1" ex="$2"
  shift 2
  # Global: the EXIT trap runs after the function has returned.
  pack_part="$(dirname -- "$out")/.$(basename -- "$out").partial"
  trap 'rm -f "$pack_part"' EXIT
  if [ -n "$ex" ]; then
    echo "total $(find "$@" -path "./$ex" -prune -o ! -type d -print | wc -l)"
    set -- --exclude="./$ex" -- "$@"
  else
    echo "total $(find "$@" ! -type d | wc -l)"
    set -- -- "$@"
  fi
  tar -czvf - "$@" 2>&1 >"$pack_part" | tar_names || return 1
  mv "$pack_part" "$out" && trap - EXIT
}
unpack() {
  case "$1" in
    *.zip)
      echo "total $(unzip -l "$1" | awk 'NF >= 4 && $1 ~ /^[0-9]+$/ && $0 !~ /\/$/' | wc -l)"
      unzip -o "$1" -d "$2" | while IFS= read -r l; do
        case $l in *"inflating: "*|*"extracting: "*) echo "${l#*: }" ;; esac
      done ;;
    *)
      exec 3<"$1"
      (tar -xvf - -C "$2" <&3 | lines) &
      report_reads $! "$(stat -c %s "$1")" ;;
  esac
}
`

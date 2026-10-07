package files

// jobFuncs start the scripts of jobs. SIGTERM (Cancel) and a lost stream (HUP, or PIPE on the next
// output) exit, so EXIT traps remove what is left half done; pipefail keeps the exit code of tar.
// The functions print what progressWriter reads:
//   - pack OUT EXCLUDE ENTRY… packs the entries ("./name") of the current folder into the tar.gz
//     OUT, without the folder EXCLUDE when it is set. It prints "total N" (files to pack) and every
//     entry tar packs: the archive goes to stdout, which makes tar write the names unbuffered. OUT
//     is written under a temporary name, so a failed or cancelled archive never exists.
//   - unpack ARCHIVE DEST extracts a tar (any compression) or zip archive into DEST. A tar reports
//     the bytes of the archive read (report_reads, fd 3), a zip the files it unpacks of those
//     unzip lists (the names a tar prints come buffered, in blocks).
//   - report_reads PID SIZE prints "pos N SIZE" every second while PID runs, N being the read
//     position of fd 3 of this shell (shared with the process that reads from it), and returns
//     the exit code of PID.
//   - lines passes whole lines on, so they never mix with the "pos" lines; tar_names passes the
//     entries tar prints ("./…") to stdout and its errors to stderr (tar writes both to stderr
//     while the archive goes to stdout).
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

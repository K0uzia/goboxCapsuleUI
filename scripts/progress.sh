#!/usr/bin/env bash
# Lance une commande en affichant un indicateur animé, le temps écoulé et la
# dernière ligne produite (ex. le paquet Go en cours de compilation).
# En cas d'échec, affiche le journal complet et renvoie le code d'erreur.
#
# Usage : scripts/progress.sh "Message" commande [args...]
set -u

msg=$1; shift
log=$(mktemp)
trap 'rm -f "$log"' EXIT

"$@" >"$log" 2>&1 &
pid=$!

# Pas de terminal (CI, redirection) : on attend simplement
if [ ! -t 1 ]; then
	echo "$msg..."
	wait "$pid"; rc=$?
	[ "$rc" -eq 0 ] || cat "$log"
	exit "$rc"
fi

frames='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏'
start=$SECONDS i=0
cols=$(tput cols 2>/dev/null || echo 80)
tput civis 2>/dev/null
trap 'tput cnorm 2>/dev/null; rm -f "$log"' EXIT
trap 'kill "$pid" 2>/dev/null; exit 130' INT TERM

while kill -0 "$pid" 2>/dev/null; do
	last=$(tail -n 1 "$log" 2>/dev/null)
	line=$(printf '%s %s (%ds) %s' "${frames:i++%${#frames}:1}" "$msg" $((SECONDS - start)) "$last")
	printf '\r\033[K\033[36m%s\033[0m' "${line:0:cols-1}"
	sleep 0.1
done

wait "$pid"; rc=$?
printf '\r\033[K'
if [ "$rc" -eq 0 ]; then
	printf '\033[32m✔\033[0m %s (%ds)\n' "$msg" $((SECONDS - start))
else
	printf '\033[31m✘\033[0m %s : échec (code %d)\n' "$msg" "$rc"
	cat "$log"
fi
exit "$rc"

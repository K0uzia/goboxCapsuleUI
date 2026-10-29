// SPDX-License-Identifier: GPL-3.0-or-later
// Hook PreToolUse (Bash, PowerShell) : refuse l'arrêt d'un processus par motif
// ou par nom, qui peut tuer le shell de la session (rule web-method « Vérifier »).
let entree = '';
process.stdin.on('data', (morceau) => { entree += morceau; });
process.stdin.on('end', () => {
    let commande = '';
    try { commande = JSON.parse(entree).tool_input?.command ?? ''; } catch { process.exit(0); }
    const interdits = [
        /\bpkill\b/i,
        /\bkillall\b/i,
        /\btaskkill\b[^;&|\n]*\s[/-]im\b/i,
        /\b(stop-process|spps)\b[^;&|\n]*\s-n(a(m(e)?)?)?\b/i,
    ];
    if (interdits.some((motif) => motif.test(commande))) {
        process.stderr.write('Refusé : arrêter un processus seulement par son propre PID (cmd & pid=$!; ...; kill "$pid"), jamais par pkill, killall, taskkill /IM ou Stop-Process -Name (rule web-method « Vérifier »).\n');
        process.exit(2);
    }
});

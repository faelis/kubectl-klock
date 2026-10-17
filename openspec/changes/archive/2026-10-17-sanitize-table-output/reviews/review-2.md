<!--
SPDX-FileCopyrightText: 2026 faelis

SPDX-License-Identifier: CC-BY-4.0
-->

# Revue 2 : sanitize-table-output

Juge : `/review`, rapport seul, aucun code modifié. Branche
`feat/sanitize-table`, diff `git diff 7944cb8...HEAD` (HEAD `084fd29`).
Toute la branche est relue, avec une attention particulière aux
corrections de la revue 1 (`a623dbf`, `084fd29`).

## Vérifications lancées

| Vérification | Résultat |
| --- | --- |
| `go build ./...`, `go vet ./...` (copie `git archive HEAD`) | OK |
| `go test ./...` (même copie) | OK, 4 paquets |
| Tests de `a623dbf` sur le code d'avant `084fd29` | 3 échouent |
| `openspec validate sanitize-table-output --strict` | valide |
| `markdownlint-cli2` sur le dossier du change | aucune erreur |
| `grep -nP` des caractères invisibles sur `git ls-files` | vide |
| Fichiers de test hors commits `test(...)` | aucun |
| 3 mutations des correctifs (copie temporaire) | 3 cassent un test |

`bound_test.go` et `sanitize_test.go` n'apparaissent que dans
`3661d06 test(klock)` et `a623dbf test(klock)`. Les commits `fix(klock)`
ne touchent aucun test.

Avant `084fd29`, les trois nouveaux tests échouent pour la bonne raison :
cellule d'erreur de 5054 caractères, U+3164 brut dans le refus de `-o`,
en-tête `A\\n` à double antislash. Mutations : rétablir `%q` dans le refus
« unknown output format », retirer `boundCell` de la cellule `<error: …>`,
retirer `sanitizeCell` des valeurs `-L`. Chaque fois, un test échoue.

Régressions cherchées : aucun test ni texte du dépôt ne dépend de l'ancien
format `%q`. `customcolumns_test.go:207` attend toujours
`unsupported output format: "json", …` et passe. Le README ne cite aucun
de ces messages. `cmd/root.go:215` affiche l'erreur telle quelle après
`ERROR:`.

## Critical

Aucun.

## Important

Aucun.

## Minor

Constats de la revue 1 corrigés : m5, m8, m9, m10, m11, m12.

Constats de la revue 1 encore ouverts, sans changement de code depuis :

- m1. `pkg/klock/klock.go:526` : valeurs `-L` échappées mais pas bornées.
- m2. `pkg/klock/sanitize_test.go:160` : `TestTableModeStatusStillStyled`
  appelle `parseCell` directement, sans passer par `addObjectToTable`.
- m3. `pkg/klock/sanitize_test.go:151` : `TestTableModeCellInteger` reste
  vert si les entiers passent par `sanitizeCell` (`klock.go:518`).
- m4. `pkg/klock/klock.go:637` : `boundCell` peut couper un échappement.
  Sonde : 1022 `a` suivis de U+3164 finissent par `aaaaaaa\u…`.
- m6. `pkg/klock/klock.go:504` : cellule `NAMESPACE` non échappée.
- m7. `pkg/klock/sanitize_test.go:58` : `viewOf` configure toujours un Pod.
  Les branches Event, Job et CronJob de `parseCell` restent sans test.

Nouveaux constats :

- n1. `pkg/klock/klock.go:94`, `:97` et `pkg/klock/customcolumns.go:70` :
  `%q` échappait `"` et `\`, `"%s"` avec `sanitizeCell` ne le fait plus.
  Le message devient ambigu. Sonde : `-o 'x"y'` donne
  `unknown output format: "x"y"`. Une saisie littérale de `x\u202e` et
  un vrai U+202E produisent le même message. L'en-tête `A"B` donne
  `column "A"B": …`. Aucun caractère de contrôle brut n'atteint le
  terminal, d'où Minor. Correctif possible : `strconv.QuoteToASCII` pour
  les refus de `Validate`, qui échappe aussi U+3164. Sinon, noter ce
  compromis dans la décision 5 du design.
- n2. `pkg/klock/klock.go:636` : la docstring de `boundCell` affirme que
  les échappements sont « counted, not split ». La sonde de m4 montre le
  contraire. Garder « counted » et retirer « not split », ou faire reculer
  la coupe au début de l'échappement.
- n3. `pkg/klock/klock.go:94` : dans la branche « unsupported »,
  `o.Output` vaut forcément l'un des noms ASCII listés. `sanitizeCell` n'y
  change rien, et aucune mutation ne peut le détecter. Le retirer, ou le
  garder par symétrie avec la ligne 97.
- n4. `openspec/changes/sanitize-table-output/design.md:49` : « échappe
  `o.Output` dans tous ses refus, entre guillemets droits ». Le refus
  `-L` (`klock.go:81`) n'a pas de guillemets, comme dans kubectl. Préciser
  que seuls les deux refus de format sont entre guillemets.
- n5. `openspec/changes/sanitize-table-output/specs/table-output/spec.md:22`
  compte 81 caractères, introduit par la coupe de phrase de `b17c68e`.
  `markdownlint` ne le signale pas (MD013 non strict : aucune espace
  après la colonne 80), mais la règle du dépôt vise 80. Renvoyer `MUST`
  à la ligne suivante.

Hors périmètre, pour mémoire : `pkg/klock/klock.go:508` place une cellule
brute du serveur dans un message d'erreur, affiché par `Table.SetError`,
exclu par la proposal. Les messages d'erreur de `-o` ne sont pas bornés
(3068 octets pour une spec de 3000 `Z`), ce que la décision 4 assume.

## Conformité aux deltas

Chaque scénario des deux deltas a un test nommé d'après lui, et ce test
passe. Les trois scénarios ajoutés depuis la revue 1 (« Message d'erreur
borné », « Format refusé avec caractère ignorable », « En-tête échappé
une seule fois dans les erreurs ») correspondent à `bound_test.go:50`,
`:68` et `:83`. Les six scénarios existants de l'exigence modifiée
restent dans le delta. Le scénario principal « Format toujours refusé »
garde son message exact. Les décisions 1 à 5 du design correspondent au
code, aux réserves n1 et n4 près.

Docs : proposal, design, tasks et specs relus contre le skill
`redaction` : pas de calque ni de tic repéré. Le README et l'ADR
viennent aux étapes release-doc et adr.

Non lancé : sous-agents spécialistes, passe adverse et QA exploratoire du
skill `/review`, remplacés par les sondes et mutations ci-dessus. Mise à
jour de gstack proposée par le préambule ignorée.

REVIEW: PASS

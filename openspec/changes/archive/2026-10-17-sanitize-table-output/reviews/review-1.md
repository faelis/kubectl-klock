<!--
SPDX-FileCopyrightText: 2026 faelis

SPDX-License-Identifier: CC-BY-4.0
-->

# Revue 1 : sanitize-table-output

Juge : `/review`, rapport seul, aucun code modifié. Branche
`feat/sanitize-table`, diff `git diff 7944cb8...HEAD` (HEAD `f1de44f`).

## Vérifications lancées

| Vérification | Résultat |
| --- | --- |
| `go build ./...`, `go vet ./pkg/klock` | OK |
| `go test ./...` (copie `git archive HEAD`) | OK, 4 paquets |
| `openspec validate sanitize-table-output --strict` | valide |
| `markdownlint-cli2` sur le dossier du change | aucune erreur |
| `grep -nP` des caractères invisibles sur `git ls-files` | vide |
| Fichiers de test hors commits `test(...)` | aucun |
| 13 mutations du code (copie temporaire) | 12 cassent un test |

Les fichiers de test (`bound_test.go`, `sanitize_test.go`) n'apparaissent
que dans `3661d06 test(klock)`. Le commit `7731e84 fix(klock)` ne touche
aucun test.

Mutations : retirer l'échappement des valeurs `-L`, des en-têtes, des
cellules du mode table, des erreurs de `-o` ou de `-L`, le test
`Other_Default_Ignorable_Code_Point`, `QuoteRuneToASCII`, l'un des deux
`boundCell`, ou inverser majuscules et échappement : chaque fois, au moins
un test échoue. Seule la mutation « échapper aussi les entiers » reste
verte (voir m3).

## Critical

Aucun.

## Important

Aucun.

## Minor

Constats déjà relevés par les étapes précédentes, confirmés :

- m1. `pkg/klock/klock.go:524` : les valeurs `-L` sont échappées mais pas
  bornées. L'API limite une valeur de label à 63 caractères, le risque
  reste théorique.
- m2. `pkg/klock/sanitize_test.go` (`TestTableModeStatusStillStyled`) :
  le test appelle `parseCell` directement. Il ne couvre pas le chemin
  `addObjectToTable` qui fournit la valeur échappée.
- m3. `pkg/klock/sanitize_test.go` (`TestTableModeCellInteger`) : le test
  vérifie seulement la présence de `3`. Faire passer `int64` par
  `sanitizeCell` (retrait du `case` en `pkg/klock/klock.go:516`) le laisse
  vert.
- m4. `pkg/klock/klock.go:634` : `boundCell` peut couper au milieu d'un
  échappement et afficher `aaaa\…`. Le test de borne n'y échappe que parce
  que 1000 + 2 × 12 tombe juste sur 1024.
- m5. `pkg/klock/klock.go:615` : la cellule `<error: …>` n'est pas bornée.
  Sonde : avec `X:.metadata.annotations.x[?(@.a)]` et une annotation de
  5000 `a`, la cellule compte 5054 caractères. Le contenu vient de l'objet,
  donc d'un tenant, et la spec `custom-columns-output` exige « chaque
  cellule » au plus 1024. L'expression doit toutefois être choisie par
  l'utilisateur ; je garde Minor, comme C4.
- m6. `pkg/klock/klock.go:502` : la cellule `NAMESPACE` n'est pas
  échappée. L'API valide les noms de namespace.
- m7. `pkg/klock/sanitize_test.go:61` : `viewOf` configure toujours un Pod.
  Aucun test ne passe par les branches Event (`LAST SEEN`), Job
  (`DURATION`) ou CronJob (`LAST SCHEDULE`) de `parseCell`.

Constat écarté : « cas U+FEFF du mode custom-columns non testé ». Le test
existe, `pkg/klock/customprinter_test.go:165`.

Nouveaux constats :

- m8. `pkg/klock/klock.go:92` et `:95` : les refus « unsupported » et
  « unknown output format » formatent `o.Output` avec `%q`. `%q` laisse
  bruts U+3164, U+115F et U+034F, que `unicode.IsPrint` tient pour
  imprimables. Sonde : `Options{Output: "x" + U+3164 + "y"}.Validate()`
  renvoie le caractère brut. La spec exige l'échappement des caractères
  ignorables dans les messages d'erreur de `-o`. La décision 5 du design
  ne traite que les caractères de contrôle. La valeur vient de
  l'utilisateur, d'où Minor. Correctif : `sanitizeCell(o.Output)` entre
  guillemets, ou `strconv.QuoteToASCII`.
- m9. `pkg/klock/customcolumns.go:69` : `%q` sur un en-tête déjà échappé
  double les barres obliques inverses. Sonde : l'en-tête `A` suivi d'un
  saut de ligne donne le message `column "A\\n": unterminated array`.
  Le défaut existait avant ; la ligne est touchée par ce change.
- m10. `pkg/klock/klock.go:608` : la docstring de `evalCell` décrit la
  jointure, `<none>` et l'échappement, mais pas la borne de 1024 runes que
  la fonction applique désormais.
- m11. `openspec/changes/sanitize-table-output/design.md:59` : « seul
  l'échappement change l'affichage » oublie la borne de 1024, qui change
  aussi l'affichage des listes et objets longs.
- m12. Rédaction : deux phrases de spec dépassent 30 mots.
  `specs/custom-columns-output/spec.md:13` (environ 35 mots) et
  `specs/table-output/spec.md:19` (environ 33 mots). Les couper en deux.

Hors périmètre, pour mémoire : `pkg/klock/klock.go:506` place une cellule
brute du serveur dans un message d'erreur. Ce message passe par
`Table.SetError`, exclu par la proposal.

## Conformité aux deltas

Chaque scénario des deux deltas a un test nommé d'après lui dans
`sanitize_test.go` ou `bound_test.go`, et ce test passe. Les six scénarios
existants de l'exigence modifiée « Échappement des séquences terminal »
sont repris dans le delta. Les décisions 1 à 5 du design correspondent au
code ; seul m8 montre une limite de la décision 5.

Docs : pas de calque ni de tic repéré dans proposal, design, tasks et
specs. Le README et l'ADR viennent aux étapes release-doc et adr.

Non lancé : sous-agents spécialistes et passe adverse du skill `/review`,
remplacés par les sondes et mutations ci-dessus. Mise à jour de gstack
proposée par le préambule ignorée.

REVIEW: PASS

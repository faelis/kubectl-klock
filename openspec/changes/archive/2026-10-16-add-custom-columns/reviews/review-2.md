<!--
SPDX-FileCopyrightText: 2026 faelis

SPDX-License-Identifier: CC-BY-4.0
-->

# Revue 2 : add-custom-columns

Branche `feat/custom-columns`, base `main` (`git diff main...HEAD`),
HEAD `c2e392c`. Juge : `/review` en rapport seul, aucune modification du
code relu. Deuxième tour, après les corrections de `9ab8f29..c2e392c`.

## Ce qui a été lancé

- `go build ./...`, `go vet ./...`, `go test ./...` : 81 tests verts.
- `gofmt -l pkg cmd` : seul `pkg/table/race_test.go`, hors branche.
- `git log --stat main..HEAD` : les fichiers `_test.go` ne changent que
  dans des commits `test(...)`. Conforme.
- Tests rouges vérifiés : dans une copie `mktemp`, `table.go` et
  `klock.go` remis à `7db2230`, `TestViewLongCellInNonFinalColumn` et
  cinq cas de `TestEvalCell` échouent. Les tests ajoutés mordent.
- Mode tableau : sonde `addObjectToTable` (`-A`, événements `ADDED`,
  `DELETED`, `ERROR`) lancée sur `main` et sur la branche. `View()`
  identique.
- Sondes `sanitizeCell` et `evalCell` dans la copie `mktemp` : cellule
  `<error: …>` avec OSC, NUL, U+0085, U+061C, U+200F, DEL, octets
  invalides, U+2028, U+200B, U+FEFF.
- Cellule de 1 200 caractères avec `\n`, terminal de 80 colonnes : pas
  de panique, deux lignes d'écran (en-tête et ligne).
- Formes `.a.b`, `a.b`, `{.a.b}`, `{a.b}` citées par le README : toutes
  acceptées, en ligne et en fichier.
- Binaire construit dans un dossier `mktemp`, `KUBECONFIG=/dev/null` :
  messages des options invalides conformes à la spec, code 1.
  `__complete pods -o ""` : trois propositions, directive `:6`.
- Exemples du README et de `cmd/root.go` passés à `zsh -c 'echo …'` :
  aucun `no matches found`.
- `openspec validate add-custom-columns --strict` : valide.
- `npx -y markdownlint-cli2 --no-globs $(git ls-files '*.md')` :
  0 problème.
- `uvx reuse lint` : seuls les fichiers non suivis de `.serena/`.
- Recherche de caractères invisibles ou de contrôle dans les fichiers
  de la branche (`grep -P`).
- Sous-agent adverse sur le diff hors tests.

Sous-agent adverse : 5 constats. Retenus : caractères `Cf`, `Zl`, `Zp`
(m1), en-têtes non assainis (m5), événement `ERROR` (déjà M4). Écartés :
« une cellule plus large que le terminal casse le rendu », réfuté, car
bubbletea 1.3.10 tronque chaque ligne à la largeur de la fenêtre
(`standard_renderer.go:241`, `ansi.Truncate`) ; « coût de
`strings.Repeat` », non mesuré et dominé par le calcul des largeurs,
antérieur à la branche.

Couverture non faite : pas de spécialistes Review Army, pas de sonde sur
un cluster (consigne), `origin/main` non rafraîchi.

## Statut des constats de review-1

| Constat | Statut | Preuve |
| --- | --- | --- |
| C1 panique cellule longue | corrigé | `pkg/table/table.go:522` |
| I1 `<error:>` non échappé | corrigé | `pkg/klock/klock.go:606` |
| I2 caractères de contrôle | corrigé | `pkg/klock/klock.go:628` |
| I3 README sous zsh | corrigé | `README.md:94`, `:143` |
| M1 `NoSpace` sur `wide` | non corrigé | `cmd/root.go:147` |
| M2 directive `:4` | corrigé | `cmd/root_test.go` |
| M3 commentaire périmé | corrigé | `pkg/klock/klock.go:309` |
| M4 événement `ERROR` | non corrigé | `pkg/klock/klock.go:568` |
| M5 assertion sans `ok` | corrigé | `customcolumns.go:55` |
| M6 `fmt.Errorf("%s")` | corrigé | `customcolumns.go:53` |
| M7 en-têtes recalculés | non corrigé | `pkg/klock/klock.go:444` |
| M8 LIST sans pagination | non corrigé | `pkg/klock/klock.go:304` |
| M9 fractions multiples | non corrigé | `pkg/klock/klock.go:644` |
| M10 tests faibles | en partie | voir m3 |
| M11 signature `Validate` | ouvert, ADR | tâche 5.6 |
| D1 trois fichiers | corrigé | `proposal.md:42-50` |
| D2 design périmé | corrigé | `docs/design/…:39-51` |
| D3 « Tous les namespaces » | corrigé | `spec.md:182` |
| D4 « whitespace », accolades | corrigé | `README.md:138`, `:151` |
| D5 trait d'union, point | corrigé | `cmd/root.go:139` |

M1, M4 et M7 à M9 avaient été classés « à laisser » par review-1.

## Critical

Aucun.

## Important

### I1. Caractère U+202E brut dans le scénario de la spec

- Fichier :
  `openspec/changes/add-custom-columns/specs/custom-columns-output/spec.md:111`.
- Problème : le troisième résultat attendu est le caractère U+202E
  lui-même (octets `e2 80 ae`, vérifié avec `od -c`), pas le texte
  `\u202e`. Le scénario exige donc l'affichage brut d'une inversion
  bidi, à l'inverse de l'exigence (« forme visible ») et du code
  (`sanitizeCell` rend `\u202e`, `customprinter_test.go` attend
  `` `\u202e` ``). Un vérificateur qui lit le scénario ne peut pas
  conclure. Dans un rendu Markdown, le caractère retourne la fin de la
  ligne : c'est le motif « Trojan Source » que ce change veut bloquer.
  Le fichier part dans `openspec/specs/` à l'archivage.
- Correction : remplacer le caractère par `` `\u202e` `` (six
  caractères ASCII). Ajouter au besoin une vérification `grep -P` des
  contrôles bidi sur les `.md`. Cause probable : l'outil d'écriture de
  l'agent convertit la séquence `\u202e` en caractère brut ; le juge a
  subi le même effet en écrivant ses sondes. Relire le fichier écrit avec
  `grep -P` ou `od -c`.
- testable : non.

## Minor

- m1. `pkg/klock/klock.go:632` : `sanitizeCell` laisse passer U+2028,
  U+2029 (séparateurs de ligne et de paragraphe) et les caractères de
  format `Cf` (U+200B, U+2060, U+FEFF, U+00AD). Sonde : `"\u2028x"` et
  `"\u200bzw"` sortent inchangés. Le sous-agent ajoute U+200D et les
  caractères d'étiquette (U+E0001 à U+E007F). Effet limité : la plupart des
  terminaux ne passent pas à la ligne sur U+2028, mais un espace de
  largeur nulle permet d'imiter une autre valeur dans une cellule.
  Confiance moyenne, à vérifier. Correction : échapper aussi
  `unicode.Is(unicode.Zl, r)`, `unicode.Zp` et `unicode.Cf`.
  testable : oui.
- m2. `pkg/klock/klock.go:503` (antérieur à la branche) : en mode
  tableau, les cellules de la `Table` du serveur ne passent pas par
  `sanitizeCell`. Un `MESSAGE` d'événement ou une colonne
  d'impression d'un CRD avec `\n` ou ESC casse toujours la mise en page.
  Hors du périmètre de la spec, qui vise le mode custom. Correction
  (change à part) : appeler `sanitizeCell` dans `parseCell`, chemin
  commun aux deux modes. testable : oui.
- m3. Tests : aucun test du mode tableau après l'extraction de
  `baseRow` et le passage de `parseCell` par `colorCell` (reste de
  M10). La sonde du juge ne trouve pas de régression. Correction : un
  test `addObjectToTable` avec `-A` et `DELETED`, dans un commit
  `test(...)`. testable : oui.
- m4. `pkg/klock/klock.go:601` : ligne de commentaire de 84 colonnes,
  les autres en font 80 au plus. Rewrapper. testable : non.
- m5. `pkg/klock/klock.go:444` : les en-têtes de la spec ne passent pas
  par `sanitizeCell`. Un ESC ou un saut de ligne dans `-o` ou
  `KLOCK_OUTPUT` arrive brut au terminal. L'utilisateur s'expose
  lui-même ; kubectl fait de même. Correction : assainir les en-têtes
  dans `ParseCustomColumns`. testable : oui.

## Documentation (skill redaction)

- D1. `openspec/changes/add-custom-columns/design.md:61-63` : la
  décision 5 dit encore « `printers.EscapeTerminal` sur chaque valeur ».
  Le code appelle `sanitizeCell`, qui échappe aussi les caractères de
  contrôle et le message `<error: …>`. Mettre la décision à jour.
  testable : non.
- D2. `docs/design/2026-10-16-custom-columns.md:22` : « Rendu des
  cellules : la valeur brute, comme kubectl ». La puce suivante (`:26`)
  dit l'inverse pour les caractères de contrôle. Écrire « la valeur,
  comme kubectl, échappée (voir ci-dessous) ». testable : non.
- D3. `openspec/changes/add-custom-columns/tasks.md:49-52` : aucune tâche
  ne cite les scénarios ajoutés après review-1 (« Cellule très
  longue », « Saut de ligne et tabulation », « Autres caractères de
  contrôle », « Message d'erreur échappé »). Les rattacher à une tâche
  (3.1 ou une tâche de correction cochée). testable : non.
- D4. `README.md:147` : « so a value cannot break the table layout ».
  Vrai pour les contrôles C0 et C1, pas pour U+2028 ni les caractères de
  largeur nulle (m1). Écrire « so a newline or tab in a value does not
  break the table layout », ou corriger m1. Le texte ne dit pas non plus
  que les messages `<error: …>` sont échappés. testable : non.

Docstrings : `sanitizeCell`, `evalCell`, `colorCell`, `baseRow`,
`customRow`, `acceptHeader` et les symboles publics (`CustomColumn`,
`ParseCustomColumns`, `Validate`, `Printer.CustomColumns`) en ont une,
exacte. Pas de code commenté ajouté. Pas de tic d'IA relevé dans le
README, le document de design ni les artefacts OpenSpec. README en
anglais, artefacts en français : conforme.

## Verdict

0 Critical, 1 Important (I1), 5 Minor, 4 constats de documentation.
Les quatre constats bloquants de review-1 sont corrigés, sans
régression du rendu de la table ni du mode tableau.

REVIEW: FAIL

<!--
SPDX-FileCopyrightText: 2026 faelis

SPDX-License-Identifier: CC-BY-4.0
-->

# Revue 3 : add-custom-columns

Branche `feat/custom-columns`, base `main` (`git diff main...HEAD`),
HEAD `7aeca8e`. Juge : `/review` en rapport seul, aucune modification du
code relu. Troisième tour, approuvé par l'utilisateur, après les
corrections de `7dc6244..7aeca8e`.

## Ce qui a été lancé

- `go build ./...`, `go vet ./...`, `go test ./...` : 88 tests, tous
  verts.
- `gofmt -l pkg cmd` : seul `pkg/table/race_test.go`, hors branche.
- `git log --stat main..HEAD` : les fichiers `_test.go` ne changent que
  dans des commits `test(...)` (`e7577a7` pour ce tour). Conforme.
- Tests rouges vérifiés : dans une copie `mktemp` (`git archive HEAD`),
  `klock.go` et `customcolumns.go` remis à `c2e392c`. `TestEvalCell` et
  `TestParseCustomColumnsHeaderEscapesANSI` échouent. Les tests mordent.
- Mutations dans la copie : retirer la cellule `NAMESPACE` de
  `addObjectToTable` fait échouer `TestPrintObjTableMode` ; retirer
  `row.MarkDeleted()` de `baseRow` le laisse vert (voir m2).
- Sondes `sanitizeCell` et `ParseCustomColumns` dans la copie : séquence
  ZWJ d'emoji, U+00AD, U+200C, U+E0041, U+0600, U+3164, U+FFA0, U+115F,
  U+034F, U+FE0E, U+FE0F ; en-tête avec ESC et U+200B suivi d'une
  expression invalide.
- `openspec validate add-custom-columns --strict` : valide.
- `npx -y markdownlint-cli2 --no-globs $(git ls-files '*.md')` :
  0 problème sur 8 fichiers.
- `grep -nP` des caractères invisibles et bidi sur `git ls-files` : vide.
- Sous-agent adverse sur le diff du tour : 3 constats, tous vérifiés par
  sonde et retenus (m1, m3, m4). Réfutés : ESC ou CR bruts (traités par
  `printers.EscapeTerminal`), contrôles C1, isolats bidi, caractères
  d'étiquette, UTF-8 invalide (remplacé par U+FFFD), marques combinantes,
  effet de l'en-tête assaini sur `jsonpath`, coût par cellule.

Couverture non faite : pas de spécialistes Review Army, pas de passe
Codex, pas de sonde sur un cluster (consigne). `cmd/` n'a pas changé
depuis review-2 : sondes du binaire non relancées.

## Statut des constats de review-2

| Constat | Statut | Preuve |
| --- | --- | --- |
| I1 U+202E brut dans la spec | corrigé | `spec.md:112`, `grep -P` vide |
| m1 `Zl`, `Zp`, `Cf` | corrigé | `pkg/klock/klock.go:635` |
| m2 mode tableau | reporté | hors périmètre, avant la branche |
| m3 test du mode tableau | en partie | `tablemode_test.go`, voir m2 |
| m4 commentaire de 84 colonnes | corrigé | `pkg/klock/klock.go:601-602` |
| m5 en-têtes non assainis | corrigé | `customcolumns.go:66` |
| D1 décision 5 périmée | corrigé | `design.md:62-66` |
| D2 « valeur brute » | corrigé | `docs/design/…:22-23` |
| D3 scénarios sans tâche | corrigé | `tasks.md:71-77` |
| D4 README trop affirmatif | corrigé | `README.md:146-149`, voir m3 |

Points reportés par l'appelant (m2 de review-2, M4, M7, M8, M9) : je
maintiens le report. Chacun existe déjà sur `main` ou ne touche que le
coût ou la couleur. Aucun n'est une régression de la branche.

## Critical

Aucun.

## Important

Aucun.

## Minor

- m1. `pkg/klock/klock.go:635` : les caractères ignorables par défaut
  hors `Cf` passent bruts. U+3164, U+FFA0 et U+115F (lettres de
  remplissage hangeul) et U+034F (CGJ, largeur nulle) ne sont ni `Cf`
  ni `Bidi_Control`. Sonde : `"Failed"` suivi de U+3164 sort inchangé ;
  la cellule ressemble à `Failed` mais `StatusColumn` ne la colore pas.
  La spec (`spec.md:90-93`) parle de « caractères invisibles » sans les
  borner, le code s'arrête à `Cf`. Correction : ajouter
  `unicode.Other_Default_Ignorable_Code_Point` à `unicode.In`, ou
  restreindre le texte de l'exigence aux catégories `Zl`, `Zp`, `Cf`.
  Confiance 7. testable : oui.
- m2. `pkg/klock/tablemode_test.go:105` : après `DELETED`, le test ne
  vérifie que `RowIndex`. Retirer `row.MarkDeleted()` le laisse vert
  (mutation lancée). Le constat m3 de review-2 demandait de garder le
  cas `DELETED`. Correction : vérifier
  `Status == table.StatusDeleted`, comme `customprinter_test.go:267`,
  dans un commit `test(...)`. testable : oui.
- m3. `pkg/klock/klock.go:635` : les sélecteurs de variante (U+FE0E,
  U+FE0F, Mn) passent bruts. `ansi.PrintableRuneWidth` leur donne une
  colonne (`"Failed"` suivi de U+FE0E mesure 7), le terminal zéro : la
  suite de la ligne se décale d'une colonne. `README.md:149` affirme
  pourtant « they cannot break the table layout ». Défaut déjà présent
  en mode tableau. Correction : échapper aussi
  `unicode.Variation_Selector`, ou nuancer la phrase du README. Rendu
  dans un vrai terminal non vérifié. Confiance 6. testable : oui.
- m4. `pkg/klock/customcolumns.go:69` : `%q` sur un en-tête déjà
  assaini double les barres obliques inverses. Sonde : en-tête `X`, ESC,
  `[1m`, U+200B, expression `.metadata.name[` ; le message montre la
  forme échappée de U+200B précédée de deux barres au lieu d'une.
  Correction : `%q` sur `col.Header` brut (`strconv.Quote` échappe déjà
  ESC et les caractères non imprimables), ou `"column \"%s\""` sur
  l'en-tête assaini. testable : oui.
- m5. `pkg/klock/tablemode_test.go:30` :
  `TestParseCustomColumnsHeaderEscapesANSI` teste l'analyse de la spec,
  pas le mode tableau. Sa place est `customcolumns_test.go`.
  testable : non.

Choix assumé, non compté : échapper `Cf` coupe les séquences d'emoji
avec ZWJ (U+200D) et le ZWNJ (U+200C) du persan ou de l'hindi, qui
s'affichent sous forme échappée. Le design (`docs/design/…:27-30`) le
revendique (« plus strict que kubectl »).

## Documentation (skill redaction)

- D1. `spec.md:122` : « la spec est `-o custom-columns=` suivi de… ».
  Ailleurs, « la spec » désigne la partie après `=` (`spec.md:33`) et
  l'option complète s'écrit « l'option est `-o …` » (`spec.md:160`).
  Écrire « l'option est ». testable : non.
- D2. `docs/design/2026-10-16-custom-columns.md:27-30` : énumération
  hétérogène (« de contrôle, de contrôle bidirectionnel, séparateur…
  et de format invisible ») dans une phrase de plus de 30 mots. Couper :
  « Après `printers.EscapeTerminal`, les caractères suivants
  s'affichent échappés : contrôles, contrôles bidi, séparateurs de ligne
  et de paragraphe (Zl, Zp), caractères de format (Cf). » testable : non.
- D3. `tasks.md:71-73` : titre `4bis`, tâche `4b.1`. Aligner les deux
  numérotations. testable : non.

Docstrings : `sanitizeCell` et `evalCell` décrivent le comportement
actuel ; le commentaire ajouté dans `ParseCustomColumns` explique le
pourquoi. Aucun nouveau symbole public. Pas de code commenté ajouté.
Pas de tic d'IA relevé. README en anglais, artefacts en français :
conforme. Lignes de commentaire de plus de 80 colonnes restantes :
`customcolumns_test.go:65`, `:117` (commit de test) et `klock.go:99`
(avant la branche), sans conséquence.

## Verdict

0 Critical, 0 Important, 5 Minor, 3 constats de documentation. Le
constat bloquant de review-2 (I1) est corrigé ; m1, m4, m5 et D1 à D4
aussi ; m3 l'est en partie (m2 ci-dessus). Pas de régression : tests
verts, mode tableau inchangé sous mutation de la colonne `NAMESPACE`.

REVIEW: PASS

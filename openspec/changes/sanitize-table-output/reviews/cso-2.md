<!--
SPDX-FileCopyrightText: 2026 faelis

SPDX-License-Identifier: CC-BY-4.0
-->

# CSO 2 : sanitize-table-output

Branche `feat/sanitize-table`, base `7944cb8` (`git diff 7944cb8...HEAD`),
HEAD `8f827d1`. Juge : `/cso --diff --code --offline`, rapport seul,
aucune modification du code audité. Aucun cluster contacté.

Run `gstack-cso` : `1792223334208-eaabbca1f179d2b6`, état `partial`.
Le helper retient `docs/demo.gif` (binaire illisible), hors surface de
sécurité. Rapport du helper :
`~/.gstack/security/cso/b4145034fd039df15d4995c8/`
`1792223334208-eaabbca1f179d2b6/report.md`.

## Périmètre et méthode

- Lecture statique par le helper : `pkg/klock/klock.go`,
  `pkg/klock/customcolumns.go`, `pkg/table/table.go`, `pkg/table/row.go`,
  `internal/util/strings.go`, `cmd/completion.go`.
- Sonde hors dépôt, dans le scratchpad (`p2/`) : elle recopie
  `sanitizeCell`, `boundCell` et `SplitsFromStart` de HEAD et pilote
  `textinput` de bubbles `v1.0.0` (frappe `s`, puis Tab). Aucun code du
  dépôt n'a été exécuté.
- Contre-vérification séquentielle, faute d'agent indépendant.
- Chaîne d'approvisionnement : `go.mod` et `go.sum` ne changent pas
  (`git diff --stat 7944cb8...HEAD -- go.mod go.sum` vide).
- Fichiers `_test.go` : modifiés uniquement dans `3661d06`, `a623dbf` et
  `370c591`, trois commits `test(klock)`. Conforme.
- Commentaires du diff : pas de code commenté, pas de symbole public
  ajouté. La docstring de `boundCell` décrit bien la coupe après
  l'échappement ; le commentaire de `baseRow` (`klock.go:566`) explique
  pourquoi `SortKey` reste brut.

## Suivi des audits précédents

| ID | État | Preuve |
| --- | --- | --- |
| C1 | fermé | cellules `klock.go:520`, en-têtes `:472`, labels `:483`, `:526` |
| C2 | fermé | `customcolumns.go:53`, `:66`, `:70` ; `klock.go:81`-`:97` |
| C3 | fermé | `QuoteRuneToASCII`, ignorables (`klock.go:666`) ; sonde |
| C4 | fermé | `boundCell` en `klock.go:520`, `:620`, `:631` |
| N1 | fermé | `Suggestion: sanitizeCell(name)` (`klock.go:569`) ; sonde |
| N2 | partiel | NAMESPACE fermé (`klock.go:504`) ; erreur `:508` ouverte |
| N3 | ouvert | Info, inchangé (`klock.go:641`) |
| N4 | ouvert | Info, inchangé (`klock.go:94`, `:97`) |
| N5 | ouvert | Info, inchangé |

### N1 : suggestions de filtre

`baseRow` échappe désormais le nom avant de le copier dans
`Row.Suggestion`. `customRow` appelle `baseRow` (`klock.go:595`) : le
mode custom-columns hérite du correctif. `SortKey` reste brut, mais
`row.go:87` ne s'en sert que pour trier.

La sonde nourrit `textinput` avec trois noms hostiles : OSC 52
(`system\x1b]52;c;aGk=\x07`), une suite de segments séparés par `-`
(ESC, `\u202e`, `\u3164`, `\u034f`, `\u009b`, `\n`, `\t`,
`\u2028`, `\u200b`) et 3000 ESC. Pour chacun, après suppression des
SGR que `textinput` émet lui-même :

- la vue après la frappe `s` ne contient aucune rune de contrôle,
  bidi, invisible ou ignorable ;
- Tab recopie la suggestion dans la valeur, qui reste en ASCII sûr
  (`system^[]52;c;aGk=\\a`, échappement tel qu'affiché par `%q`) ;
- l'écho de la ligne d'état (`"%s" (%d/%d rows)`, `table.go:446`) reste
  sûr lui aussi.

`SplitsFromStart` coupe sur l'octet `-`, qui ne peut pas tomber au
milieu d'une rune : chaque préfixe reste un texte échappé. Les tests
`TestTableModeSuggestionEscaped` et `TestCustomRowSuggestionEscaped`
(commit `370c591`) couvrent les deux modes.

### Chemins bruts restants

Sources serveur tracées jusqu'au terminal : cellules, en-têtes, labels,
NAMESPACE, suggestions, filtre, message de colonnes en trop. Seul ce
dernier reste brut (O1). Le filtre tapé à la main peut contenir un
octet brut collé par l'opérateur ; il vient de lui-même, sans
franchir de frontière. `cmd/completion.go` reprend la complétion de
kubectl, hors diff.

## Constats

| ID | Sév. | Conf. | Lieu | Impact |
| --- | --- | --- | --- | --- |
| O1 | Low | haute | `klock.go:508` | ESC brut, serveur hostile |
| O2 | Info | haute | `klock.go:569` | suggestion non bornée |
| N3 | Info | haute | `klock.go:641` | mémoire x16 avant la borne |
| N4 | Info | haute | `klock.go:94`, `:97` | guillemets ambigus |
| N5 | Info | haute | `klock.go:660` | U+2800 brut, cellule vide |

### O1 (Low, préexistant, hors périmètre) : erreur de colonnes en trop

Reste de N2. `klock.go:508` place la cellule brute et `p.colDefs` dans
l'erreur. Elle remonte par `errorChan` (`klock.go:250`) jusqu'à
`SetError`, et `View` rend `m.err.Error()` tel quel (`table.go:457`).
Un serveur API authentique construit des lignes cohérentes ; il faut un
kubeconfig piégé ou un `APIService` agrégé hostile. La proposal exclut
`SetError` de son périmètre. Correction : échapper et borner dans
`Table.SetError`, point unique pour toutes les erreurs.

### O2 (Info) : suggestion échappée mais non bornée

`Suggestion` passe par `sanitizeCell` sans `boundCell`. Sonde : un nom
de 3000 ESC donne une valeur de 6001 runes, en ASCII sûr. Aucun effet
sur le terminal. Au-delà de 1024 runes, la suggestion acceptée ne
correspond plus à la cellule NAME bornée, et le filtre n'affiche plus la
ligne. Piste : `boundCell(sanitizeCell(name))`.

### N3, N4, N5 (Info)

Inchangés depuis CSO 1. N3 : `sanitizeCell` traite la chaîne entière
avant `boundCell`. N4 : le format `"%s"` laisse un `"` brut, venu de
l'utilisateur. N5 : U+2800 et les runes non attribuées passent bruts,
usurpation visuelle seule, à mentionner avec C5 dans le README.

## Verdict

N1 est fermé dans les deux modes ; C1 à C4 restent fermés. Aucun constat
Critical, High ni Medium ouvert dans le périmètre du change. O1 relève
de `SetError`, exclu par la proposal ; il mérite un change de suivi.

CSO: PASS

<!--
SPDX-FileCopyrightText: 2026 faelis

SPDX-License-Identifier: CC-BY-4.0
-->

# CSO 1 : sanitize-table-output

Branche `feat/sanitize-table`, base `7944cb8` (`git diff 7944cb8...HEAD`),
HEAD `aaa1cb0`. Juge : `/cso --diff --code --offline`, rapport seul,
aucune modification du code audité. Aucun cluster contacté.

Run `gstack-cso` : `1792222887987-e40621d22843fa08`, état `partial`.
Le helper retient `docs/demo.gif` (binaire illisible), hors surface de
sécurité. Rapport du helper :
`~/.gstack/security/cso/b4145034fd039df15d4995c8/`
`1792222887987-e40621d22843fa08/report.md`.

## Périmètre et méthode

- Lecture statique : `pkg/klock/klock.go`, `pkg/klock/customcolumns.go`,
  `pkg/table/table.go`, historique `7944cb8..HEAD`.
- Sources des dépendances lues dans le cache Go : bubbles `v1.0.0`
  (`textinput`), apimachinery et cli-runtime `v0.37.0`.
- Sondes hors dépôt, dans le scratchpad : un programme Go recopie
  `sanitizeCell` et `boundCell` de HEAD (tables Unicode 17.0.0) et
  pilote `textinput` de bubbles. Aucun code du dépôt n'a été exécuté.
- Contre-vérification séquentielle, faute d'agent indépendant.
- Chaîne d'approvisionnement : `go.mod` et `go.sum` ne changent pas
  (`git diff --stat 7944cb8...HEAD -- go.mod go.sum` vide).
- Fichiers `_test.go` : modifiés uniquement dans `3661d06` et `a623dbf`,
  deux commits `test(klock)`. Conforme.
- Commentaires du diff : pas de code commenté, pas de symbole public
  ajouté ; `boundCell` et `maxCellRunes` ont leur commentaire.

## Suivi de l'audit précédent

| ID | État | Preuve |
| --- | --- | --- |
| C1 | fermé | cellules `klock.go:520`, en-têtes `:472`, labels `:483`, `:526` |
| C2 | fermé | `customcolumns.go:53`, `:70` ; `klock.go:81`, `:94`, `:97` |
| C3 | fermé | sonde : U+3164, U+FFA0, U+115F, U+1160, U+034F, U+17B4 échappés |
| C4 | fermé | `boundCell` en `klock.go:520`, `:618`, `:629` |
| C5 | ouvert | 260 sélecteurs de variante bruts ; README à `release-doc` |

- C1 : toute cellule qui n'est ni `nil`, ni booléen, ni nombre passe par
  `sanitizeCell` puis `boundCell` avant `parseCell`. Les branches qui
  renvoient la cellule telle quelle reçoivent donc un texte sûr.
  L'en-tête est échappé après `strings.ToUpper`, ce qui garde
  `\u202e` en minuscules.
- C2 : la sonde montre que `strconv.QuoteRuneToASCII` échappe aussi les
  ignorables imprimables, que `%q` laissait passer.
- C4 : la coupe à 1024 runes tombe après l'échappement. Elle peut couper
  une séquence en `\u20…`, mais ce reste est de l'ASCII : aucune rune
  brute ne réapparaît (sonde).

## Constats

| ID | Sév. | Conf. | Lieu | Impact |
| --- | --- | --- | --- | --- |
| N1 | Medium | moyenne | `klock.go:567` | ESC brut via suggestions |
| N2 | Low | haute | `klock.go:504`, `:508` | ESC brut, serveur hostile |
| N3 | Info | haute | `klock.go:637` | mémoire x16 avant la borne |
| N4 | Info | haute | `klock.go:97` | guillemets ambigus |
| N5 | Info | haute | `klock.go:657` | U+2800 brut, cellule vide |

### N1 (Medium, préexistant, hors diff) : suggestions de filtre brutes

`baseRow` copie `metadata.name` brut dans `Row.Suggestion`
(`klock.go:567`). `updateFilterSuggestions` le passe à
`SetSuggestions` (`table.go:229`, `:240`), qui ne nettoie rien. La vue
de `textinput` rend le reste de la suggestion tel quel
(`textinput.go:797`). La touche Tab l'ajoute à la valeur sans passer par
le nettoyeur (`textinput.go:564`). Le filtre s'affiche ensuite brut dans
la ligne d'état (`table.go:446`) et dans le champ (`table.go:425`).

- Source : la proposal pose que l'API valide les noms. C'est faux pour
  RBAC. Les noms `Role`, `RoleBinding`, `ClusterRole` suivent la règle
  de segment de chemin, qui n'interdit que `/` et `%`
  (`apimachinery@v0.37.0/pkg/api/validate/content/path.go:37`). Le
  validateur RBAC lui-même n'est pas dans le cache Go ; je le déduis des
  noms comme `system:controller:…`, refusés par la règle DNS. D'où la
  confiance moyenne.
- Exploitation : un admin de namespace crée un `Role` nommé `system`
  suivi de `ESC ] 52 ; c ; <base64> BEL`. L'opérateur lance
  `klock roles -A`, tape `/` puis `s`. Le terminal reçoit la séquence ;
  s'il accepte OSC 52, le presse-papiers change. Tab recopie la
  séquence dans la ligne d'état.
- Sonde : avec bubbles `v1.0.0`, `View()` contient ESC et BEL après la
  frappe `s`, et Tab donne la valeur `system\x1b]52;c;…\a`.
- La cellule NAME est échappée par le diff ; seule la suggestion fuit.
  Les deux modes sont touchés, car `customRow` passe aussi par `baseRow`.
- Correction : `Suggestion: sanitizeCell(name)` dans `baseRow`. Un seul
  point couvre le mode tableau et le mode custom-columns.
- Testable : oui (`PrintObj` avec un nom contenant ESC, puis
  `Row.Suggestion` sans octet ESC).

### N2 (Low, préexistant, hors diff) : réponse d'un serveur hostile

Deux chemins restent bruts, mais un serveur API authentique ne les
alimente pas.

- `klock.go:504` ajoute `GetNamespace()` sans échappement ni borne. Le
  serveur valide les noms de namespace (label DNS).
- `klock.go:508` place la cellule et `p.colDefs` bruts dans l'erreur.
  L'erreur remonte par `errorChan` (`klock.go:250`) jusqu'à `SetError`,
  et `View` rend `m.err.Error()` tel quel (`table.go:457`). Les messages
  `Status` du serveur suivent le même chemin.
- Exploitation : un kubeconfig pointe vers un serveur piégé, ou un
  `APIService` agrégé répond. La `Table` renvoyée porte une cellule de
  trop qui contient ESC ; la ligne d'erreur l'affiche brute.
- La proposal exclut `SetError` de son périmètre ; review-2 le note.
- Correction : échapper et borner dans `Table.SetError`, point unique
  pour toutes les erreurs, et échapper la cellule NAMESPACE.
- Testable : oui.

### N3 (Info) : amplification mémoire avant la borne

`sanitizeCell` traite la chaîne entière avant `boundCell`, qui la
convertit entière en `[]rune` (`klock.go:638`). Sonde : 1 Mio de NUL
donne 4 Mio échappés, puis 16 Mio de runes, le temps d'un événement.
L'API borne un objet à environ 1,5 Mio et seule la cellule bornée reste
en mémoire. Pas d'impact durable. Piste : compter avec
`utf8.RuneCountInString` et couper par index d'octet, ou couper
l'entrée à `maxCellRunes` runes avant l'échappement, qui ne fait
qu'allonger le texte.

### N4 (Info) : guillemets ambigus dans les erreurs `-o` (n1 de review-2)

Le format `"%s"` (`klock.go:94`, `:97`, `customcolumns.go:70`) laisse un
`"` brut dans la valeur. Sonde : `-o 'x", allowed formats are: json"'`
affiche `unknown output format: "x", allowed formats are: json"",
allowed formats are: wide`. La valeur vient de l'utilisateur lui-même,
sans effet sur l'état du terminal. `customcolumns.go:57` garde un `%q`,
mais ce chemin est inatteignable en pratique.

### N5 (Info) : U+2800 et runes non attribuées

U+2800 (blanc braille), U+0378 (non attribué) et U+E000 (usage privé)
passent bruts (sonde). Ils ne sont pas « default ignorable » ; U+2800
s'affiche comme un blanc. Une cellule qui n'en contient que paraît vide,
comme avec U+3164 avant C3. Usurpation visuelle seule. À mentionner avec
C5 dans le README, sans changer le code.

## Verdict

C1 à C4 sont fermés ; C5 reste à documenter à l'étape `release-doc`.
Aucun constat Critical ni High dans le diff. N1 est préexistant et hors
des lignes du diff, mais il contredit une prémisse de la proposal ; il
mérite un change de suivi.

CSO: PASS

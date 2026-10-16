<!--
SPDX-FileCopyrightText: 2026 faelis

SPDX-License-Identifier: CC-BY-4.0
-->

# CSO 1 : add-custom-columns

Branche `feat/custom-columns`, base `main` (`git diff main...HEAD`),
HEAD `5dd45ed`. Juge : `/cso --diff --code --offline`, rapport seul,
aucune modification du code audité. Aucun cluster contacté.

Run `gstack-cso` : `1792175500362-51ba16fdf30c8215`, état `partial`.
Le helper a retenu `docs/demo.gif` (binaire illisible) ; ce fichier
n'entre pas dans la surface de sécurité. Rapport du helper :
`~/.gstack/security/cso/b4145034fd039df15d4995c8/`
`1792175500362-51ba16fdf30c8215/report.md`.

## Périmètre et méthode

- Lecture statique par le helper : `pkg/klock/customcolumns.go`,
  `pkg/klock/klock.go`, `pkg/table/table.go`, `pkg/table/row.go`,
  `cmd/root.go`, historique `main..HEAD`.
- Sources des dépendances lues dans le cache Go : kubectl, client-go et
  cli-runtime `v0.37.0`, bubbletea `v1.3.10`.
- Sonde hors dépôt : un programme Go à part reprend le prédicat de
  `sanitizeCell` (tables Unicode 17.0.0). Aucun code du dépôt n'a été
  exécuté.
- Contre-vérification séquentielle, faute d'agent indépendant.
- Chaîne d'approvisionnement : `go.mod` et `go.sum` ne changent pas sur
  la branche (`git diff --stat main...HEAD -- go.mod go.sum` vide).
- Fichiers `_test.go` : modifiés uniquement dans des commits
  `test(...)`. Conforme.

## Contrôles vérifiés sans constat

- Valeurs de cellule : `evalCell` passe chaque valeur par
  `sanitizeCell` (`klock.go:614`). ESC devient `^[`, puis C0, C1
  (dont U+009B CSI), `Bidi_Control`, `Zl`, `Zp` et `Cf` sont échappés.
  Un octet UTF-8 invalide ressort en U+FFFD.
- Messages `<error: …>` : assainis (`klock.go:606`). C'est utile :
  `jsonpath.go:503` recopie la valeur de l'objet dans l'erreur.
- En-têtes : assainis à l'analyse (`customcolumns.go:66`).
- Erreurs de `baseRow` : seule `time.Parse` reprend une donnée du
  serveur, et `time.ParseError` l'échappe déjà.
- Fichier de spec : kubectl l'ouvre (`customcolumn_flags.go:90`) et lit
  deux lignes avec `bufio.Scanner`. Chaque ligne est bornée à 64 Kio. Le
  chemin vient de l'utilisateur, sans changement de privilège. Une FIFO
  ou `/dev/stdin` bloque, au choix de l'utilisateur.
- Objets complets sans `as=Table` (`klock.go:740`) : même comportement
  que `kubectl get -o custom-columns`. `klock secrets -o
  custom-columns=…` charge `.data` en mémoire, mais ne l'affiche que si
  l'expression JSONPath le sélectionne. klock n'expose pas les options
  `klog` et n'écrit rien sur disque.
- Pagination : `RequestChunksOf` reste commenté, comme sur `main`. En
  mode custom-columns, la liste initiale pèse plus lourd (objets
  complets), coût mémoire local lancé par l'utilisateur. Pas de
  constat.

## Constats

| ID | Sév. | Conf. | Lieu | Impact |
| --- | --- | --- | --- | --- |
| C1 | Medium | moyenne | `klock.go:730` | échappements bruts, mode tableau |
| C2 | Low | haute | `customcolumns.go:69` | ESC brut sur stderr |
| C3 | Low | haute | `klock.go:634` | usurpation visuelle |
| C4 | Low | moyenne | `klock.go:617` | coût CPU, cellule énorme |
| C5 | Info | haute | `klock.go:634` | décalage de colonnes |

### C1 (Medium, préexistant, hors diff) : mode tableau non assaini

`addObjectToTable` (`klock.go:511`) passe chaque cellule de la `Table`
serveur à `parseCell`. Sa branche par défaut appelle `colorCell`
(`klock.go:730`) sans `sanitizeCell`. `columnsView` (`table.go:525`)
rend la chaîne telle quelle. bubbletea tronque les lignes avec
`ansi.Truncate`, qui conserve les séquences. kubectl, lui, échappe ses
cellules (`printers.WriteEscaped`).

- Exploitation : un tenant crée un Event dont le message contient
  `ESC ] 52 ; c ; <base64> BEL`. L'admin lance `klock events`. Un
  terminal qui accepte OSC 52 remplace le presse-papiers. L'admin colle
  ensuite une commande piégée. Même chemin pour une colonne
  `additionalPrinterColumns` d'une CR.
- Le diff ne crée pas ce défaut : il réorganise `addObjectToTable` et
  `parseCell` sans changer l'affichage. Il le rend plus visible, car le
  mode custom-columns est assaini et le mode par défaut ne l'est pas.
- Correction : appliquer `sanitizeCell` à toute cellule chaîne dans
  `parseCell`, et aux colonnes de labels (`klock.go:515`).
- Testable : oui (`PrintObj` en mode tableau, cellule avec ESC).
- Non bloquant pour ce diff ; à traiter dans un change séparé.

### C2 (Low) : erreurs JSONPath non assainies

`ParseCustomColumns` assainit l'en-tête mais enveloppe l'erreur du
parseur avec `%w` (`customcolumns.go:69`). Le parseur recopie le texte
brut de l'expression : `invalid array index %s` (`parser.go:306`),
`unquote string %s` (`parser.go:417`). L'erreur de kubectl remontée en
`customcolumns.go:53` reprend aussi la spec brute
(`customcolumn.go:84`).

- Exploitation : un `columns.txt` partagé contient `.a[<ESC>c]` en
  ligne 2. `klock pods -o custom-columns-file=columns.txt` affiche
  `invalid array index <ESC>c` sur stderr, et le terminal se réinitialise.
- Limite : l'utilisateur choisit le fichier, et kubectl affiche la même
  erreur brute.
- Correction : `sanitizeCell(err.Error())` aux lignes 53 et 69.
- Testable : oui (l'erreur ne doit contenir aucun octet ESC).

### C3 (Low) : caractères ignorables hors `Cf` (m1 de review-3)

La sonde confirme : U+3164, U+FFA0, U+115F, U+1160, U+034F et U+17B4
passent bruts. Ils relèvent de `Other_Default_Ignorable_Code_Point`, pas
de `Cf`.

- Exploitation : une annotation ou un `status.message` ne contient que
  des U+3164. La cellule paraît vide alors qu'elle porte du contenu, ou
  `Failed` suivi de U+3164 échappe à la coloration d'erreur.
- Impact : usurpation visuelle seule, sans effet sur l'état du terminal.
  `metadata.name`, validé par l'API, ne peut pas les contenir.
- Correction : ajouter `unicode.Other_Default_Ignorable_Code_Point` au
  test de `sanitizeCell`.
- Testable : oui.

### C4 (Low) : longueur de cellule non bornée

`evalCell` (`klock.go:617`) renvoie la valeur entière. À chaque mise à
jour, `updateColumnWidths` (`table.go:529`) mesure toute la cellule ;
`ansi.Truncate` la parcourt à chaque rendu.

- Exploitation : `klock configmaps -o custom-columns=D:.data`, avec une
  ConfigMap d'environ 1 Mio écrite par un tiers. L'interface ralentit.
- Limite : l'utilisateur choisit la colonne ; l'API borne la taille d'un
  objet (environ 1,5 Mio) ; bubbletea tronque à la largeur de la
  fenêtre. Le processus reste utilisable.
- Correction : tronquer la cellule rendue (par exemple 1024 runes et
  `…`).
- Testable : oui.

### C5 (Info) : sélecteurs de variante (m3 de review-3)

U+FE0E, U+FE0F et U+E0100 passent bruts (`Variation_Selector`, `Mn`).
L'écart de largeur décale la suite de la ligne. C'est un défaut de mise
en page, sans effet sur l'état du terminal ni sur le contenu affiché.
Les échapper casserait l'affichage des emoji ; je ne le recommande pas
pour la sécurité. Nuancer la phrase du README suffit. Testable : oui.

## Verdict

Aucun constat Critical ni High dans le diff. C1 est préexistant et hors
périmètre ; il mérite un change de suivi. C2 à C4 peuvent attendre ou
entrer dans le même change.

CSO: PASS

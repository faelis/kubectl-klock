<!--
SPDX-FileCopyrightText: 2026 faelis

SPDX-License-Identifier: CC-BY-4.0
-->

# Revue 1 : add-custom-columns

Branche `feat/custom-columns`, base `main` (`git diff main...HEAD`),
HEAD `7db2230`. Juge : `/review` en rapport seul, aucune modification du
code relu.

## Ce qui a été lancé

- `go build ./...`, `go vet ./...`, `go test ./...` : 75 tests verts.
- `gofmt -l pkg cmd` : seul `pkg/table/race_test.go`, hors branche.
- Binaire construit dans un dossier `mktemp` : messages d'erreur des
  options invalides conformes à la spec (spec vide, colonne sans
  expression, JSONPath invalide, fichier introuvable, nombre différent,
  `-L`, `-o json`, `KLOCK_OUTPUT`).
- `__complete pods -o ""` : trois propositions, directive `:6`.
- Sondes Go dans une copie `mktemp` du dépôt (cellule longue, retour à la
  ligne, caractères de contrôle, cellule `<error: …>`).
- Sous-agent adverse : 7 constats ; les points 1 à 3 ont été reproduits
  par le juge avant d'entrer dans ce rapport.
- `git log --stat main..HEAD` : les fichiers `_test.go` ne changent que
  dans des commits `test(...)`. Conforme.

Couverture non faite : pas de spécialistes Review Army, pas de sonde sur
un cluster (l'e2e `kind` est déjà vert), `origin/main` non rafraîchi.

## Critical

### C1. Cellule longue : panique de l'interface

- Fichier : `pkg/table/table.go:524`, atteint par
  `pkg/klock/klock.go:612`.
- Problème : `lotsOfSpaces` fait 200 espaces.
  `lotsOfSpaces[:spacing]` panique dès qu'une colonne autre que la
  dernière dépasse environ 198 caractères. Le défaut existait déjà, mais
  le mode custom le rend facile à atteindre : annotations,
  `.spec.containers[*].image`, messages de conditions. Toute personne
  qui peut annoter un objet surveillé peut faire quitter klock.
- Preuve : `slice bounds out of range [:252] with length 200` dans
  `View()`, avec une annotation de 250 caractères en première colonne.
- Correction : écrire `strings.Repeat(" ", spacing)` (ou agrandir le
  tampon à la demande) à la place de la tranche fixe.
- testable : oui.

## Important

### I1. Cellule `<error: …>` non échappée

- Fichier : `pkg/klock/klock.go:604`.
- Problème : `fmt.Sprintf("<error: %v>", err)` ne passe pas par
  `EscapeTerminal`. client-go met la valeur de l'objet dans l'erreur
  (`jsonpath.go:503`, `"%v is not array or slice and cannot be
  filtered"`). Une annotation `\x1b]0;…\x07` arrive brute au terminal
  (titre, OSC 52, effacement d'écran). La spec exige de neutraliser les
  séquences d'échappement dans les cellules.
- Preuve : sonde avec `F:.metadata.annotations[?(@.x=="y")]`, la cellule
  contient l'octet `0x1b`.
- Correction : passer le texte d'erreur par le même assainissement que
  les valeurs (voir I2).
- testable : oui.

### I2. Assainissement limité à ESC et CR

- Fichier : `pkg/klock/klock.go:612`.
- Problème : `printers.EscapeTerminal` ne remplace que `\x1b` et `\r`.
  Un `\n` coupe la ligne en deux dans la table (sonde : la ligne `p0`
  occupe deux lignes d'écran). `\t`, `\b`, BEL, NUL, U+009B (CSI C1) et
  U+202E (inversion bidi) passent tels quels. `\b` peut réécrire la
  cellule voisine. Déclencheurs réels : `.metadata.annotations` avec
  `last-applied-configuration`, données de ConfigMap. Le skeptic l'avait
  classé faible « comme kubectl ». Dans une TUI paginée ligne à ligne,
  l'effet est plus grave que dans la sortie de kubectl.
- Correction : après `EscapeTerminal`, remplacer chaque rune
  `unicode.IsControl` et les contrôles bidi (U+202A à U+202E, U+2066 à
  U+2069) par une forme visible (`\n`, `\t`, `?`). Ajouter un scénario
  à la spec.
- testable : oui.

### I3. Exemple du README cassé sous zsh

- Fichier : `README.md:142`.
- Problème : `custom-columns=…,IMAGE:.spec.containers[0].image` n'est
  pas cité. zsh y voit un motif de fichier et répond
  `no matches found` (vérifié avec `zsh -c`). zsh est le shell par défaut
  de macOS. Bash avec `failglob` échoue aussi.
- Correction : citer l'argument,
  `-o 'custom-columns=NAME:.metadata.name,IMAGE:.spec.containers[0].image'`.
  Citer aussi `README.md:94` et `cmd/root.go:86` par cohérence.
- testable : non.

## Minor

- M1. `cmd/root.go:147` : `ShellCompDirectiveNoSpace` s'applique aussi à
  `wide`, donc pas d'espace après `-o wide`. Après
  `custom-columns-file=`, la complétion repropose les trois littéraux et
  jamais un fichier. Correction : filtrer selon `toComplete` et rendre
  `NoSpace` seulement pour les préfixes en `=`. testable : oui.
- M2. `cmd/root_test.go:43` : le commentaire cite la directive `":4"`,
  la sortie réelle est `:6`. Le test ne vérifie pas la directive.
  Correction dans un commit `test(...)`. testable : non.
- M3. `pkg/klock/klock.go:307` : le commentaire « Validate already
  guarantees Output is a valid value » ne décrit plus la closure, qui lit
  `len(w.Printer.CustomColumns)` et non `Output`. Le retirer. testable :
  non.
- M4. `pkg/klock/klock.go:384` et `:537` : un événement `ERROR` du watch
  (410 Expired, Forbidden) donne `metadata.uid: want string, got <nil>`.
  Le message du serveur disparaît. La branche `watch.Error` de `baseRow`
  (`:565`) ne s'exécute jamais. Le mode tableau masquait déjà ce message
  autrement. Correction : sur `watch.Error`, renvoyer l'erreur tirée du
  `Status` (`apierrors.FromObject`). testable : oui.
- M5. `pkg/klock/customcolumns.go:54` : assertion de type sans
  `comma-ok`, panique si kubectl change son type de retour. testable :
  oui.
- M6. `pkg/klock/customcolumns.go:52` : `fmt.Errorf("%s", …)` sans
  verbe utile ; `errors.New` suffit. testable : non.
- M7. `pkg/klock/klock.go:442` : `printCustom` reconstruit les en-têtes
  et relance `SetHeaders` (recalcul des largeurs) à chaque événement.
  Les calculer une fois. Même coût qu'en mode tableau. testable : non.
- M8. `pkg/klock/klock.go:308` (confiance moyenne, à vérifier) : en mode
  custom, la LIST renvoie les objets complets sans pagination
  (`RequestChunksOf` reste commenté ligne 302) et `resource.Result` les
  garde en mémoire tant que le watch tourne. Non mesuré. Piste :
  `RequestChunksOf(500)` comme kubectl. testable : non.
- M9. `pkg/klock/klock.go:628` : une valeur multiple comme `0/1,1/1`
  passe par `StatusColumn` et n'a pas la couleur de fraction. Acceptable,
  à noter. testable : oui.
- M10. Tests faibles : `pkg/klock/customprinter_test.go:197` (cas
  supprimé : seule la couleur d'avertissement est exclue, une chaîne
  simple n'est pas exigée). Aucun test du mode tableau après l'extraction
  de `baseRow`. La relecture du juge et du sous-agent ne trouve pas de
  régression. testable : oui.
- M11. `pkg/klock/klock.go:71` : `Options.Validate` change de signature
  dans un paquet exporté. À citer dans l'ADR et les notes de version.
  testable : non.

## Documentation (skill redaction)

- D1. `openspec/changes/add-custom-columns/proposal.md:42` et `:48` :
  « touche deux fichiers de code ». La branche en touche trois :
  `pkg/klock/customcolumns.go` est nouveau. Corriger le compte et la
  liste. testable : non.
- D2. `docs/design/2026-10-16-custom-columns.md` :
  `:35-39` cite `NewCustomColumnsPrinterFromSpec` et `…FromTemplate`, le
  code appelle `NewCustomColumnsPrintFlags().ToPrinter`.
  `:40` cite `transformRequests`, qui n'existe plus (`acceptHeader`).
  `:45` dit que les en-têtes sont posés « une fois », le code les pose à
  chaque événement. `:78` : trois fichiers de code, pas deux. Ajouter
  une ligne « écarts à l'implémentation » ou corriger. testable : non.
- D3. `specs/custom-columns-output/spec.md:155` (dossier du change) :
  « Toutes les namespaces ». L'usage français de Kubernetes met
  « namespace » au masculin : « Tous les namespaces ». testable : non.
- D4. `README.md:147` : « separated by blanks » ; l'anglais courant dit
  « separated by whitespace ». Le README ne dit pas que la forme
  `{.metadata.name}` est acceptée aussi (scénario « Expression entre
  accolades »). testable : non.
- D5. `cmd/root.go:139` : « comma separated » prend un trait d'union, et
  la virgule qui relie les deux phrases (« pairs, FILE holds ») doit
  devenir un point. testable : non.

Docstrings : tous les symboles publics ajoutés en ont une
(`CustomColumn`, `ParseCustomColumns`, `Validate`,
`Printer.CustomColumns`). Pas de code commenté ajouté ; l'ancien
`transformRequests` commenté a disparu. Pas de tic d'IA relevé dans le
README ni dans les artefacts.

## Tri des constats mineurs de `progress.md`

- À corriger avant fusion : le README sous zsh (I3), la cellule
  `<error: …>` non échappée (montée en I1), `\n` et `\t` bruts (L1 du
  verify, montés en I2).
- À corriger si peu coûteux : assertion de type (M5), commentaire `:4`
  et directive non testée (M2), test du cas supprimé (M10).
- Laisser ou remettre à plus tard : `errors.New` (M6), en-têtes recalculés
  (M7), fractions multiples (M9), indirection `check` de
  `request_test.go`, déplacement du déballage dans `colorCell`, « There
  is no --no-headers » (utile, à garder), test d'`Execute` (couvert par
  l'e2e).
- Déjà résolu : détection du mode par `strings.HasPrefix` (groupe 2),
  remplacée par `len(Printer.CustomColumns) > 0`. Il en reste le
  commentaire périmé (M3).
- Hors périmètre : `q` ne quitte pas (antérieur à la branche).

## Verdict

1 Critical (C1), 3 Important (I1, I2, I3), 11 Minor, 5 constats de
documentation.

REVIEW: FAIL

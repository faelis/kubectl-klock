<!--
SPDX-FileCopyrightText: 2026 faelis

SPDX-License-Identifier: CC-BY-4.0
-->

# Design : sanitize-table-output

Taille S. Motivation : voir `proposal.md`, section Why.

## Context

`sanitizeCell` (`pkg/klock/klock.go`) échappe déjà les cellules et
en-têtes du mode custom-columns. En mode table, `addObjectToTable`
transmet chaque cellule du serveur à `parseCell`. Plusieurs branches de
`parseCell` renvoient la cellule telle quelle (`return cell`), et la
branche par défaut passe par `colorCell` sans échappement.
`updateColDefHeaders` et `labelColumnHeader` ne font que passer les noms
en majuscules. Les valeurs de labels sont recopiées sans traitement.

## Decisions

1. **Échapper à l'entrée du mode table.** Dans `addObjectToTable`, avant
   `parseCell`, toute cellule qui n'est ni `nil`, ni un nombre, ni un
   booléen devient `boundCell(sanitizeCell(fmt.Sprint(cell)))`. Toutes les
   branches de `parseCell` reçoivent alors du texte sûr, et un seul point
   suffit. Échapper dans `colorCell` seulement laisserait passer les
   branches `return cell` (durée de Job, `LAST SEEN`, `RESTARTS`).
2. **En-têtes.** `sanitizeCell(strings.ToUpper(name))` dans
   `updateColDefHeaders` et `labelColumnHeader`. L'échappement vient après
   les majuscules : sinon `\u202e` deviendrait `\U202E`. Les valeurs de
   labels passent aussi par `sanitizeCell`.
3. **Caractères ignorables.** `sanitizeCell` ajoute
   `unicode.Other_Default_Ignorable_Code_Point` à son test et remplace
   `strconv.QuoteRune` par `strconv.QuoteRuneToASCII`. Vérifié avec Go :
   `QuoteRune` laisse U+3164, U+115F et U+034F bruts, car `unicode.IsPrint`
   les tient pour imprimables. `QuoteRuneToASCII` donne `\u3164`. Pour les
   runes déjà échappées (`\n`, `\a`, `\u009b`, `\u202e`), la sortie est la
   même.
4. **Longueur.** Nouvelle fonction interne `boundCell(s string) string` :
   au-delà de `maxCellRunes = 1024` points de code, elle garde les 1024
   premiers et ajoute `…`. `evalCell` l'applique au résultat joint et à la
   cellule `<error: …>`, le mode table à chaque cellule texte. Une fonction à part plutôt que dans
   `sanitizeCell` : `evalCell` échappe chaque valeur avant la jointure, et
   la borne porte sur la cellule entière. Les en-têtes et les messages
   d'erreur de `-o` ne sont pas bornés.
5. **Messages d'erreur de `-o`.** `ParseCustomColumns` passe le texte des
   erreurs de kubectl et du parseur JSONPath par `sanitizeCell`.
   `Options.Validate` échappe `o.Output` dans tous ses refus, entre
   guillemets droits. `%q` ne suffit pas : il laisse U+3164 brut, car Go le
   tient pour imprimable. Pour la même raison, l'en-tête déjà échappé
   s'insère entre guillemets sans `%q`, qui doublerait les antislashs.

## Risks / Trade-offs

- Un texte coréen qui utilise U+3164 (remplissage Hangul) s'affiche
  échappé. Ce caractère sert surtout à simuler un blanc ; le compromis est
  accepté.
- La borne masque la fin des valeurs très longues ; `…` le signale.
  `-o custom-columns` permet de cibler une sous-partie de la valeur.
- `fmt.Sprint` sur une liste ou un objet donne le même texte qu'avant :
  seuls l'échappement et la borne changent l'affichage.

## Journalisation

Un seul niveau, comme avant : klock n'a pas de journal, et le changement
n'ajoute aucun traitement en arrière-plan. Aucun chemin silencieux :
chaque caractère échappé reste visible dans la cellule, et la troncature
est marquée par `…`. Les options invalides arrêtent klock avec
`ERROR: <message>`.

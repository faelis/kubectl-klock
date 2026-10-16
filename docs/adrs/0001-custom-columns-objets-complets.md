<!--
SPDX-FileCopyrightText: 2026 faelis

SPDX-License-Identifier: CC-BY-4.0
-->

# ADR-0001 : Mode custom-columns avec objets complets

## Statut

Acceptée, 2026-10-16.

## Contexte

`kubectl get -o custom-columns` permet de choisir ses colonnes par
JSONPath. klock refuse ce format. Pour suivre un champ absent de la table
renvoyée par le serveur (image d'un conteneur, nœud, annotation), il faut
revenir à `kubectl get --watch` et perdre l'interface de klock.

Toutes les requêtes de klock portent `Accept: as=Table`. Le serveur renvoie
alors les seules métadonnées : `.spec` et `.status` manquent.
`Printer.PrintObj` décode uniquement une `Table` et échoue sur un objet
ordinaire.

kubectl fournit déjà le parsing de `-o custom-columns=` et de
`-o custom-columns-file=`. Son évaluation par objet n'est pas exportée.

## Décision

En mode custom-columns, klock fait quatre choses.

1. Il demande les objets complets, sans `as=Table`. `acceptHeader(true)`
   renvoie `""`. Il évalue lui-même chaque expression JSONPath, comme
   `kubectl get -o custom-columns`.
2. Il réutilise `get.NewCustomColumnsPrintFlags().ToPrinter` pour le
   parsing. La syntaxe et les messages d'erreur restent ceux de kubectl.
3. Il assainit plus strictement que kubectl. `sanitizeCell` applique
   `printers.EscapeTerminal`, puis échappe les contrôles, les contrôles
   bidi, les séparateurs de ligne et de paragraphe (Zl, Zp) et les
   caractères de format (Cf). Cela vaut pour les valeurs, les messages
   `<error: …>` et les en-têtes. Un `\n` brut casserait la mise en page de
   l'interface.
4. Il refuse `-L` avec custom-columns, comme kubectl. Il n'ajoute pas de
   colonne `NAMESPACE` implicite avec `-A`.

Alternatives écartées :

- Lancer `CustomColumnsPrinter.PrintObj` dans un tampon, puis redécouper
  sa sortie sur les tabulations. Ce découpage est fragile face aux
  séquences d'échappement, aux valeurs vides et aux en-têtes répétés.
- Réécrire le parsing de la spec et la lecture du fichier. Une trentaine de
  lignes en double, et des messages d'erreur qui divergeraient de kubectl.
- Garder `as=Table` avec `includeObject=Object`. Le serveur calculerait une
  table inutile, et les API qui ne savent pas produire de table
  demanderaient un repli.

## Conséquences

Effets positifs :

- Le parsing reste celui de kubectl. Les messages d'erreur suivent kubectl.
- Toute expression JSONPath fonctionne, y compris sur un champ absent de la
  table du serveur.
- Une erreur d'évaluation reste dans sa cellule. Le watch continue.

Effets négatifs :

- Le mode custom-columns charge les objets complets, plus lourds que la
  table du serveur.
- `Options.Validate` change de signature et renvoie
  `([]CustomColumn, error)`. Ce paquet est exporté (constat M11).

Points ouverts, à traiter dans un change de suivi :

- Le mode tableau n'est pas assaini. Constat CSO C1, Medium, préexistant
  à cette branche.
- Les caractères `Other_Default_Ignorable_Code_Point` ne sont pas échappés.
- La longueur d'une cellule n'est pas bornée.
- Il n'y a pas de pagination. `RequestChunksOf` reste commenté, comme sur
  `main`.

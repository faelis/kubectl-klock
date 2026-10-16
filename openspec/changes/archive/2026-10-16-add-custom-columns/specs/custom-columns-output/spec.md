<!--
SPDX-FileCopyrightText: 2026 faelis

SPDX-License-Identifier: CC-BY-4.0
-->

# Spec Delta

## Purpose

Afficher en direct les ressources surveillées par klock selon des colonnes
choisies par l'utilisateur en JSONPath, avec la syntaxe de
`kubectl get -o custom-columns`.

## ADDED Requirements

### Requirement: Spec de colonnes inline

klock SHALL accepter `-o custom-columns=SPEC`, où `SPEC` est une liste
`EN-TÊTE:expression` séparée par des virgules. Les en-têtes de la table
MUST être ceux de la spec, dans l'ordre donné.

#### Scenario: Deux colonnes inline

- **WHEN** l'utilisateur lance `kubectl klock pods -o
  custom-columns=NAME:.metadata.name,NODE:.spec.nodeName` et que le pod
  `nginx` tourne sur le nœud `node-1`
- **THEN** la table a pour en-têtes `NAME`, `NODE`
- **AND** la ligne de `nginx` contient les cellules `nginx`, `node-1`

#### Scenario: Expression entre accolades

- **WHEN** la spec est `NAME:{.metadata.name}`
- **THEN** la colonne `NAME` affiche le nom de l'objet, comme avec
  `NAME:.metadata.name`

### Requirement: Fichier de colonnes

klock SHALL accepter `-o custom-columns-file=CHEMIN`. Le fichier contient
les en-têtes séparés par des blancs sur la première ligne, et les
expressions séparées par des blancs sur la seconde.

#### Scenario: Fichier valide

- **WHEN** le fichier contient `NAME   NODE` puis
  `.metadata.name   .spec.nodeName`
- **THEN** la table a pour en-têtes `NAME`, `NODE`
- **AND** la ligne du pod `nginx` sur `node-1` contient `nginx`, `node-1`

### Requirement: Valeur des cellules

Chaque cellule MUST contenir le résultat de l'expression JSONPath évaluée
sur l'objet complet. Plusieurs résultats MUST être joints par `,`. Un
résultat vide MUST s'afficher `<none>`.

#### Scenario: Champ absent

- **WHEN** la colonne est `NODE:.spec.nodeName` et que le pod n'a pas
  encore de nœud
- **THEN** la cellule `NODE` affiche `<none>`

#### Scenario: Valeurs multiples

- **WHEN** la colonne est `C:.spec.containers[*].name` et que le pod a les
  conteneurs `app` et `sidecar`
- **THEN** la cellule `C` affiche `app,sidecar`

#### Scenario: Champ hors de la table du serveur

- **WHEN** la colonne est `IMAGE:.spec.containers[0].image` et que le
  premier conteneur utilise `nginx:1.27`
- **THEN** la cellule `IMAGE` affiche `nginx:1.27`

#### Scenario: Erreur d'évaluation sur un objet

- **WHEN** la colonne est `X:.metadata.name[0]` et que le pod `nginx` est
  reçu
- **THEN** la cellule `X` commence par `<error:`
- **AND** les autres lignes et le watch continuent

#### Scenario: Cellule très longue

- **WHEN** la première de deux colonnes vaut une chaîne de 300 caractères
  `a`
- **THEN** l'affichage de la table réussit, sans panique
- **AND** il contient la seconde colonne de la ligne

### Requirement: Échappement des séquences terminal

klock MUST neutraliser les séquences d'échappement terminal, les
caractères de contrôle et les caractères invisibles présents dans les
cellules (valeurs comme messages d'erreur) et dans les en-têtes. Ces
caractères MUST s'afficher sous une forme visible.

#### Scenario: Annotation avec séquence ANSI

- **WHEN** la colonne est `A:.metadata.annotations.x` et que l'annotation
  `x` vaut l'octet `0x1b` suivi de `[31mrouge`
- **THEN** la cellule affiche `^[[31mrouge`
- **AND** elle ne contient pas l'octet `0x1b`

#### Scenario: Saut de ligne et tabulation

- **WHEN** la colonne est `A:.metadata.annotations.x` et que l'annotation
  `x` vaut `l1`, un saut de ligne, `l2`, une tabulation, `t`
- **THEN** la cellule affiche les six caractères littéraux `l1\nl2\tt`
- **AND** elle ne contient aucun caractère de contrôle

#### Scenario: Autres caractères de contrôle

- **WHEN** l'annotation `x` contient `U+0007`, `U+009B` ou `U+202E`
- **THEN** la cellule affiche respectivement `\a`, `\u009b`, `\u202e`

#### Scenario: Caractères invisibles

- **WHEN** l'annotation `x` contient `U+200B`, `U+2028` ou `U+FEFF`
- **THEN** la cellule affiche respectivement `\u200b`, `\u2028`,
  `\ufeff`

#### Scenario: En-tête avec séquence ANSI

- **WHEN** l'option est `-o custom-columns=` suivie de `X`, de l'octet
  `0x1b`, de `[31m:.metadata.name`
- **THEN** l'en-tête de la colonne est `X^[[31m`
- **AND** il ne contient pas l'octet `0x1b`

#### Scenario: Message d'erreur échappé

- **WHEN** la colonne est `X:.metadata.annotations.x[?(@.a)]` et que
  l'annotation `x` vaut l'octet `0x1b` suivi de `[31m`
- **THEN** la cellule commence par `<error:`
- **AND** elle ne contient pas l'octet `0x1b`

### Requirement: Suivi en direct

En mode custom-columns, klock SHALL mettre la table à jour à chaque
événement du watch, avec les mêmes règles que le mode tableau pour l'ajout,
la modification et la suppression.

#### Scenario: Modification

- **WHEN** le pod `nginx` affiché avec `NODE` = `<none>` reçoit un
  événement `MODIFIED` où `.spec.nodeName` vaut `node-1`
- **THEN** la ligne de `nginx` affiche `node-1`, sans ligne en double

#### Scenario: Suppression

- **WHEN** un événement `DELETED` arrive pour le pod `nginx`
- **THEN** la ligne de `nginx` est grisée comme une ligne supprimée
- **AND** elle disparaît après le délai de `--hide-deleted`

### Requirement: Objets complets demandés au serveur

En mode custom-columns, klock MUST demander les objets complets et non la
représentation `Table` du serveur. Hors de ce mode, la requête ne change
pas.

#### Scenario: En-tête Accept en mode custom-columns

- **WHEN** l'option est `-o custom-columns=NAME:.metadata.name`
- **THEN** l'en-tête `Accept` des requêtes ne contient pas `as=Table`

#### Scenario: En-tête Accept en mode tableau

- **WHEN** l'option `-o` est vide ou vaut `wide`
- **THEN** l'en-tête `Accept` des requêtes contient
  `application/json;as=Table;v=v1;g=meta.k8s.io`

### Requirement: Coloration selon la valeur

Les cellules custom MUST recevoir la même coloration selon la valeur que le
mode tableau (fractions, statuts), sauf sur une ligne supprimée.

#### Scenario: Fraction incomplète

- **WHEN** une cellule custom vaut `0/1` sur une ligne non supprimée
- **THEN** elle a le style `StyleFractionWarning`

#### Scenario: Statut d'erreur

- **WHEN** une cellule custom vaut `CrashLoopBackOff` sur une ligne non
  supprimée
- **THEN** elle a le style `StyleStatusError`

#### Scenario: Ligne supprimée

- **WHEN** une cellule custom vaut `0/1` dans un événement `DELETED`
- **THEN** elle n'a pas de style de fraction

### Requirement: Pas de colonne NAMESPACE implicite

Avec `--all-namespaces`, klock MUST NOT ajouter de colonne `NAMESPACE` en
mode custom-columns. Les lignes MUST rester triées par
`namespace/nom`.

#### Scenario: Tous les namespaces

- **WHEN** l'utilisateur lance `kubectl klock pods -A -o
  custom-columns=NAME:.metadata.name` avec `b/x` et `a/y`
- **THEN** la table a pour seul en-tête `NAME`
- **AND** la ligne `y` précède la ligne `x`

### Requirement: Options invalides refusées au démarrage

klock MUST refuser une option custom-columns invalide avant d'ouvrir
l'interface : klock affiche `ERROR:`, une espace, puis le message, et sort
avec le code 1.

#### Scenario: Spec vide

- **WHEN** l'option est `-o custom-columns=` ou `-o custom-columns`
- **THEN** le message est
  `custom-columns format specified but no custom columns given`

#### Scenario: Colonne sans expression

- **WHEN** l'option est `-o custom-columns=NAME`
- **THEN** le message est
  `unexpected custom-columns spec: NAME, expected <header>:<json-path-expr>`

#### Scenario: Expression JSONPath invalide

- **WHEN** l'option est `-o custom-columns=NAME:.metadata.name[`
- **THEN** klock sort en erreur avant l'interface
- **AND** le message contient `NAME`

#### Scenario: Fichier introuvable

- **WHEN** l'option est `-o custom-columns-file=/nexiste/pas`
- **THEN** le message commence par
  `error reading template /nexiste/pas, open /nexiste/pas:`

#### Scenario: En-têtes et expressions en nombre différent

- **WHEN** le fichier contient `NAME NODE` puis `.metadata.name`
- **THEN** le message est
  `number of headers (2) and field specifications (1) don't match`

#### Scenario: Combinaison avec -L

- **WHEN** l'utilisateur lance `kubectl klock pods -L app -o
  custom-columns=NAME:.metadata.name`
- **THEN** le message est `--label-columns option cannot be used with
  custom-columns=NAME:.metadata.name printer`

#### Scenario: Format toujours refusé

- **WHEN** l'option est `-o json`
- **THEN** le message est `unsupported output format: "json", allowed
  formats are: wide, custom-columns=..., custom-columns-file=...`

### Requirement: Complétion de -o

La complétion shell de `-o` SHALL proposer `wide`, `custom-columns=` et
`custom-columns-file=`.

#### Scenario: Complétion

- **WHEN** l'utilisateur demande la complétion de `-o`
- **THEN** les propositions sont `wide`, `custom-columns=`,
  `custom-columns-file=`

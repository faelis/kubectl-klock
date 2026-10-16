<!--
SPDX-FileCopyrightText: 2026 faelis

SPDX-License-Identifier: CC-BY-4.0
-->

# Colonnes personnalisées dans klock

Date : 2026-10-16. Statut : validé en idéation.

## Besoin

`kubectl get -o custom-columns=…` choisit les colonnes affichées par des
expressions JSONPath. klock refuse aujourd'hui ce format
(`Options.Validate`, `pkg/klock/klock.go`). L'utilisateur veut suivre en
direct les mêmes colonnes qu'avec kubectl.

## Décisions

- Formats acceptés : `-o custom-columns=SPEC` et
  `-o custom-columns-file=PATH`.
- Rendu des cellules : la valeur brute, comme kubectl, plus la coloration
  déjà calculée d'après la valeur (fractions `1/2`, statuts `Running`,
  `Error`…). Les lignes supprimées restent grisées. Les horodatages restent
  bruts.
- `-L` avec custom-columns : erreur, comme kubectl. Un label se lit déjà
  avec `.metadata.labels.<clé>`.
- `-A` avec custom-columns : pas de colonne `NAMESPACE` ajoutée, comme
  kubectl. Le tri interne reste `namespace/nom`.

## Approche retenue

klock demande les objets complets et évalue lui-même le JSONPath.

1. Le parsing des colonnes reprend celui de kubectl :
   `get.NewCustomColumnsPrinterFromSpec` et
   `get.NewCustomColumnsPrinterFromTemplate` (paquet
   `k8s.io/kubectl/pkg/cmd/get`, déjà en dépendance). klock garde seulement
   le champ `Columns`.
2. En mode custom-columns, `transformRequests` n'envoie plus l'en-tête
   `Accept: as=Table`. La liste initiale et le watch renvoient alors des
   `*unstructured.Unstructured` complets. kubectl fait de même
   (`get.go`, `transformRequests`).
3. `Printer.PrintObj` choisit son chemin. Le mode tableau ne change pas. Le
   mode custom pose les en-têtes `Column.Header` une fois, puis évalue chaque
   colonne avec `client-go/util/jsonpath` : `AllowMissingKeys(true)`,
   valeurs jointes par `,`, `<none>` quand rien ne correspond. Les
   expressions sont parsées une seule fois.
4. L'extraction des métadonnées (`uid`, `name`, `namespace`,
   `creationTimestamp`) sort de `addObjectToTable` dans une fonction commune.
   Le tri, le filtre et `--hide-deleted` marchent ainsi dans les deux modes.

## Approches écartées

- Lancer `CustomColumnsPrinter.PrintObj` dans un tampon et redécouper la
  sortie sur les tabulations : fragile face aux séquences d'échappement, aux
  valeurs vides et aux en-têtes répétés.
- Garder `as=Table` avec `includeObject=Object` : le serveur calcule une
  table inutile, et les API sans table demandent un repli.

## Erreurs et journalisation

klock n'a pas de journal : les erreurs passent par `Table.SetError`, puis le
watch redémarre. Un seul niveau suffit donc. Une spec invalide, un fichier
illisible ou la combinaison avec `-L` arrêtent klock avant l'ouverture de
l'interface. Une erreur d'évaluation sur un objet s'affiche dans la cellule
et ne coupe pas le watch.

## Tests

Tests unitaires : parsing des options (spec, fichier, valeur vide, `-L`) et
`PrintObj` en mode custom sur des objets unstructured (champ absent, valeurs
multiples, suppression, en-têtes). Pas de test e2e : le CI n'a pas de
cluster.

## Taille

S : deux fichiers de code (`pkg/klock/klock.go`, `cmd/root.go`), le README
et des tests.

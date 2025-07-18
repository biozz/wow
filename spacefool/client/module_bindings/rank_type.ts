// THIS FILE IS AUTOMATICALLY GENERATED BY SPACETIMEDB. EDITS TO THIS FILE
// WILL NOT BE SAVED. MODIFY TABLES IN YOUR MODULE SOURCE CODE INSTEAD.

// This was generated using spacetimedb cli version 1.2.0 (commit fb41e50eb73573b70eea532aeb6158eaac06fae0).

/* eslint-disable */
/* tslint:disable */
// @ts-nocheck
import {
  AlgebraicType,
  AlgebraicValue,
  BinaryReader,
  BinaryWriter,
  ConnectionId,
  DbConnectionBuilder,
  DbConnectionImpl,
  Identity,
  ProductType,
  ProductTypeElement,
  SubscriptionBuilderImpl,
  SumType,
  SumTypeVariant,
  TableCache,
  TimeDuration,
  Timestamp,
  deepEqual,
  type CallReducerFlags,
  type DbContext,
  type ErrorContextInterface,
  type Event,
  type EventContextInterface,
  type ReducerEventContextInterface,
  type SubscriptionEventContextInterface,
} from "@clockworklabs/spacetimedb-sdk";
// A namespace for generated variants and helper functions.
export namespace Rank {
  // These are the generated variant types for each variant of the tagged union.
  // One type is generated per variant and will be used in the `value` field of
  // the tagged union.
  export type Six = { tag: "Six" };
  export type Seven = { tag: "Seven" };
  export type Eight = { tag: "Eight" };
  export type Nine = { tag: "Nine" };
  export type Ten = { tag: "Ten" };
  export type Jack = { tag: "Jack" };
  export type Queen = { tag: "Queen" };
  export type King = { tag: "King" };
  export type Ace = { tag: "Ace" };

  // Helper functions for constructing each variant of the tagged union.
  // ```
  // const foo = Foo.A(42);
  // assert!(foo.tag === "A");
  // assert!(foo.value === 42);
  // ```
  export const Six = { tag: "Six" };
  export const Seven = { tag: "Seven" };
  export const Eight = { tag: "Eight" };
  export const Nine = { tag: "Nine" };
  export const Ten = { tag: "Ten" };
  export const Jack = { tag: "Jack" };
  export const Queen = { tag: "Queen" };
  export const King = { tag: "King" };
  export const Ace = { tag: "Ace" };

  export function getTypeScriptAlgebraicType(): AlgebraicType {
    return AlgebraicType.createSumType([
      new SumTypeVariant("Six", AlgebraicType.createProductType([])),
      new SumTypeVariant("Seven", AlgebraicType.createProductType([])),
      new SumTypeVariant("Eight", AlgebraicType.createProductType([])),
      new SumTypeVariant("Nine", AlgebraicType.createProductType([])),
      new SumTypeVariant("Ten", AlgebraicType.createProductType([])),
      new SumTypeVariant("Jack", AlgebraicType.createProductType([])),
      new SumTypeVariant("Queen", AlgebraicType.createProductType([])),
      new SumTypeVariant("King", AlgebraicType.createProductType([])),
      new SumTypeVariant("Ace", AlgebraicType.createProductType([])),
    ]);
  }

  export function serialize(writer: BinaryWriter, value: Rank): void {
      Rank.getTypeScriptAlgebraicType().serialize(writer, value);
  }

  export function deserialize(reader: BinaryReader): Rank {
      return Rank.getTypeScriptAlgebraicType().deserialize(reader);
  }

}

// The tagged union or sum type for the algebraic type `Rank`.
export type Rank = Rank.Six | Rank.Seven | Rank.Eight | Rank.Nine | Rank.Ten | Rank.Jack | Rank.Queen | Rank.King | Rank.Ace;

export default Rank;


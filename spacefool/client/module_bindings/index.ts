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

// Import and reexport all reducer arg types
import { Attack } from "./attack_reducer.ts";
export { Attack };
import { ClientConnected } from "./client_connected_reducer.ts";
export { ClientConnected };
import { CreateLobby } from "./create_lobby_reducer.ts";
export { CreateLobby };
import { Defend } from "./defend_reducer.ts";
export { Defend };
import { IdentityDisconnected } from "./identity_disconnected_reducer.ts";
export { IdentityDisconnected };
import { JoinLobby } from "./join_lobby_reducer.ts";
export { JoinLobby };
import { LeaveLobby } from "./leave_lobby_reducer.ts";
export { LeaveLobby };
import { PassTurn } from "./pass_turn_reducer.ts";
export { PassTurn };
import { SendMessage } from "./send_message_reducer.ts";
export { SendMessage };
import { SetName } from "./set_name_reducer.ts";
export { SetName };
import { StartGame } from "./start_game_reducer.ts";
export { StartGame };
import { TakeCards } from "./take_cards_reducer.ts";
export { TakeCards };
import { UpdateGameSettings } from "./update_game_settings_reducer.ts";
export { UpdateGameSettings };

// Import and reexport all table handle types
import { DrawTableHandle } from "./draw_table.ts";
export { DrawTableHandle };
import { GameTableHandle } from "./game_table.ts";
export { GameTableHandle };
import { GameSettingsTableHandle } from "./game_settings_table.ts";
export { GameSettingsTableHandle };
import { LobbyTableHandle } from "./lobby_table.ts";
export { LobbyTableHandle };
import { MessageTableHandle } from "./message_table.ts";
export { MessageTableHandle };
import { PlayerCardTableHandle } from "./player_card_table.ts";
export { PlayerCardTableHandle };
import { RoundTableHandle } from "./round_table.ts";
export { RoundTableHandle };
import { TurnTableHandle } from "./turn_table.ts";
export { TurnTableHandle };
import { UserTableHandle } from "./user_table.ts";
export { UserTableHandle };

// Import and reexport all types
import { Card } from "./card_type.ts";
export { Card };
import { CardLocation } from "./card_location_type.ts";
export { CardLocation };
import { DeckSize } from "./deck_size_type.ts";
export { DeckSize };
import { Draw } from "./draw_type.ts";
export { Draw };
import { DrawStatus } from "./draw_status_type.ts";
export { DrawStatus };
import { Game } from "./game_type.ts";
export { Game };
import { GameSettings } from "./game_settings_type.ts";
export { GameSettings };
import { GameStatus } from "./game_status_type.ts";
export { GameStatus };
import { Lobby } from "./lobby_type.ts";
export { Lobby };
import { LobbyStatus } from "./lobby_status_type.ts";
export { LobbyStatus };
import { Message } from "./message_type.ts";
export { Message };
import { PlayerCard } from "./player_card_type.ts";
export { PlayerCard };
import { PlayerStatus } from "./player_status_type.ts";
export { PlayerStatus };
import { Rank } from "./rank_type.ts";
export { Rank };
import { Round } from "./round_type.ts";
export { Round };
import { RoundStatus } from "./round_status_type.ts";
export { RoundStatus };
import { Suit } from "./suit_type.ts";
export { Suit };
import { Turn } from "./turn_type.ts";
export { Turn };
import { TurnStatus } from "./turn_status_type.ts";
export { TurnStatus };
import { User } from "./user_type.ts";
export { User };

const REMOTE_MODULE = {
  tables: {
    draw: {
      tableName: "draw",
      rowType: Draw.getTypeScriptAlgebraicType(),
      primaryKey: "id",
      primaryKeyInfo: {
        colName: "id",
        colType: Draw.getTypeScriptAlgebraicType().product.elements[0].algebraicType,
      },
    },
    game: {
      tableName: "game",
      rowType: Game.getTypeScriptAlgebraicType(),
      primaryKey: "id",
      primaryKeyInfo: {
        colName: "id",
        colType: Game.getTypeScriptAlgebraicType().product.elements[0].algebraicType,
      },
    },
    game_settings: {
      tableName: "game_settings",
      rowType: GameSettings.getTypeScriptAlgebraicType(),
      primaryKey: "lobbyId",
      primaryKeyInfo: {
        colName: "lobbyId",
        colType: GameSettings.getTypeScriptAlgebraicType().product.elements[0].algebraicType,
      },
    },
    lobby: {
      tableName: "lobby",
      rowType: Lobby.getTypeScriptAlgebraicType(),
      primaryKey: "id",
      primaryKeyInfo: {
        colName: "id",
        colType: Lobby.getTypeScriptAlgebraicType().product.elements[0].algebraicType,
      },
    },
    message: {
      tableName: "message",
      rowType: Message.getTypeScriptAlgebraicType(),
    },
    player_card: {
      tableName: "player_card",
      rowType: PlayerCard.getTypeScriptAlgebraicType(),
      primaryKey: "id",
      primaryKeyInfo: {
        colName: "id",
        colType: PlayerCard.getTypeScriptAlgebraicType().product.elements[0].algebraicType,
      },
    },
    round: {
      tableName: "round",
      rowType: Round.getTypeScriptAlgebraicType(),
      primaryKey: "id",
      primaryKeyInfo: {
        colName: "id",
        colType: Round.getTypeScriptAlgebraicType().product.elements[0].algebraicType,
      },
    },
    turn: {
      tableName: "turn",
      rowType: Turn.getTypeScriptAlgebraicType(),
      primaryKey: "id",
      primaryKeyInfo: {
        colName: "id",
        colType: Turn.getTypeScriptAlgebraicType().product.elements[0].algebraicType,
      },
    },
    user: {
      tableName: "user",
      rowType: User.getTypeScriptAlgebraicType(),
      primaryKey: "identity",
      primaryKeyInfo: {
        colName: "identity",
        colType: User.getTypeScriptAlgebraicType().product.elements[0].algebraicType,
      },
    },
  },
  reducers: {
    attack: {
      reducerName: "attack",
      argsType: Attack.getTypeScriptAlgebraicType(),
    },
    client_connected: {
      reducerName: "client_connected",
      argsType: ClientConnected.getTypeScriptAlgebraicType(),
    },
    create_lobby: {
      reducerName: "create_lobby",
      argsType: CreateLobby.getTypeScriptAlgebraicType(),
    },
    defend: {
      reducerName: "defend",
      argsType: Defend.getTypeScriptAlgebraicType(),
    },
    identity_disconnected: {
      reducerName: "identity_disconnected",
      argsType: IdentityDisconnected.getTypeScriptAlgebraicType(),
    },
    join_lobby: {
      reducerName: "join_lobby",
      argsType: JoinLobby.getTypeScriptAlgebraicType(),
    },
    leave_lobby: {
      reducerName: "leave_lobby",
      argsType: LeaveLobby.getTypeScriptAlgebraicType(),
    },
    pass_turn: {
      reducerName: "pass_turn",
      argsType: PassTurn.getTypeScriptAlgebraicType(),
    },
    send_message: {
      reducerName: "send_message",
      argsType: SendMessage.getTypeScriptAlgebraicType(),
    },
    set_name: {
      reducerName: "set_name",
      argsType: SetName.getTypeScriptAlgebraicType(),
    },
    start_game: {
      reducerName: "start_game",
      argsType: StartGame.getTypeScriptAlgebraicType(),
    },
    take_cards: {
      reducerName: "take_cards",
      argsType: TakeCards.getTypeScriptAlgebraicType(),
    },
    update_game_settings: {
      reducerName: "update_game_settings",
      argsType: UpdateGameSettings.getTypeScriptAlgebraicType(),
    },
  },
  versionInfo: {
    cliVersion: "1.2.0",
  },
  // Constructors which are used by the DbConnectionImpl to
  // extract type information from the generated RemoteModule.
  //
  // NOTE: This is not strictly necessary for `eventContextConstructor` because
  // all we do is build a TypeScript object which we could have done inside the
  // SDK, but if in the future we wanted to create a class this would be
  // necessary because classes have methods, so we'll keep it.
  eventContextConstructor: (imp: DbConnectionImpl, event: Event<Reducer>) => {
    return {
      ...(imp as DbConnection),
      event
    }
  },
  dbViewConstructor: (imp: DbConnectionImpl) => {
    return new RemoteTables(imp);
  },
  reducersConstructor: (imp: DbConnectionImpl, setReducerFlags: SetReducerFlags) => {
    return new RemoteReducers(imp, setReducerFlags);
  },
  setReducerFlagsConstructor: () => {
    return new SetReducerFlags();
  }
}

// A type representing all the possible variants of a reducer.
export type Reducer = never
| { name: "Attack", args: Attack }
| { name: "ClientConnected", args: ClientConnected }
| { name: "CreateLobby", args: CreateLobby }
| { name: "Defend", args: Defend }
| { name: "IdentityDisconnected", args: IdentityDisconnected }
| { name: "JoinLobby", args: JoinLobby }
| { name: "LeaveLobby", args: LeaveLobby }
| { name: "PassTurn", args: PassTurn }
| { name: "SendMessage", args: SendMessage }
| { name: "SetName", args: SetName }
| { name: "StartGame", args: StartGame }
| { name: "TakeCards", args: TakeCards }
| { name: "UpdateGameSettings", args: UpdateGameSettings }
;

export class RemoteReducers {
  constructor(private connection: DbConnectionImpl, private setCallReducerFlags: SetReducerFlags) {}

  attack(gameId: bigint, card: Card, target: Identity) {
    const __args = { gameId, card, target };
    let __writer = new BinaryWriter(1024);
    Attack.getTypeScriptAlgebraicType().serialize(__writer, __args);
    let __argsBuffer = __writer.getBuffer();
    this.connection.callReducer("attack", __argsBuffer, this.setCallReducerFlags.attackFlags);
  }

  onAttack(callback: (ctx: ReducerEventContext, gameId: bigint, card: Card, target: Identity) => void) {
    this.connection.onReducer("attack", callback);
  }

  removeOnAttack(callback: (ctx: ReducerEventContext, gameId: bigint, card: Card, target: Identity) => void) {
    this.connection.offReducer("attack", callback);
  }

  onClientConnected(callback: (ctx: ReducerEventContext) => void) {
    this.connection.onReducer("client_connected", callback);
  }

  removeOnClientConnected(callback: (ctx: ReducerEventContext) => void) {
    this.connection.offReducer("client_connected", callback);
  }

  createLobby(name: string, maxPlayers: number) {
    const __args = { name, maxPlayers };
    let __writer = new BinaryWriter(1024);
    CreateLobby.getTypeScriptAlgebraicType().serialize(__writer, __args);
    let __argsBuffer = __writer.getBuffer();
    this.connection.callReducer("create_lobby", __argsBuffer, this.setCallReducerFlags.createLobbyFlags);
  }

  onCreateLobby(callback: (ctx: ReducerEventContext, name: string, maxPlayers: number) => void) {
    this.connection.onReducer("create_lobby", callback);
  }

  removeOnCreateLobby(callback: (ctx: ReducerEventContext, name: string, maxPlayers: number) => void) {
    this.connection.offReducer("create_lobby", callback);
  }

  defend(gameId: bigint, turnId: bigint, card: Card) {
    const __args = { gameId, turnId, card };
    let __writer = new BinaryWriter(1024);
    Defend.getTypeScriptAlgebraicType().serialize(__writer, __args);
    let __argsBuffer = __writer.getBuffer();
    this.connection.callReducer("defend", __argsBuffer, this.setCallReducerFlags.defendFlags);
  }

  onDefend(callback: (ctx: ReducerEventContext, gameId: bigint, turnId: bigint, card: Card) => void) {
    this.connection.onReducer("defend", callback);
  }

  removeOnDefend(callback: (ctx: ReducerEventContext, gameId: bigint, turnId: bigint, card: Card) => void) {
    this.connection.offReducer("defend", callback);
  }

  onIdentityDisconnected(callback: (ctx: ReducerEventContext) => void) {
    this.connection.onReducer("identity_disconnected", callback);
  }

  removeOnIdentityDisconnected(callback: (ctx: ReducerEventContext) => void) {
    this.connection.offReducer("identity_disconnected", callback);
  }

  joinLobby(lobbyId: bigint) {
    const __args = { lobbyId };
    let __writer = new BinaryWriter(1024);
    JoinLobby.getTypeScriptAlgebraicType().serialize(__writer, __args);
    let __argsBuffer = __writer.getBuffer();
    this.connection.callReducer("join_lobby", __argsBuffer, this.setCallReducerFlags.joinLobbyFlags);
  }

  onJoinLobby(callback: (ctx: ReducerEventContext, lobbyId: bigint) => void) {
    this.connection.onReducer("join_lobby", callback);
  }

  removeOnJoinLobby(callback: (ctx: ReducerEventContext, lobbyId: bigint) => void) {
    this.connection.offReducer("join_lobby", callback);
  }

  leaveLobby() {
    this.connection.callReducer("leave_lobby", new Uint8Array(0), this.setCallReducerFlags.leaveLobbyFlags);
  }

  onLeaveLobby(callback: (ctx: ReducerEventContext) => void) {
    this.connection.onReducer("leave_lobby", callback);
  }

  removeOnLeaveLobby(callback: (ctx: ReducerEventContext) => void) {
    this.connection.offReducer("leave_lobby", callback);
  }

  passTurn(gameId: bigint) {
    const __args = { gameId };
    let __writer = new BinaryWriter(1024);
    PassTurn.getTypeScriptAlgebraicType().serialize(__writer, __args);
    let __argsBuffer = __writer.getBuffer();
    this.connection.callReducer("pass_turn", __argsBuffer, this.setCallReducerFlags.passTurnFlags);
  }

  onPassTurn(callback: (ctx: ReducerEventContext, gameId: bigint) => void) {
    this.connection.onReducer("pass_turn", callback);
  }

  removeOnPassTurn(callback: (ctx: ReducerEventContext, gameId: bigint) => void) {
    this.connection.offReducer("pass_turn", callback);
  }

  sendMessage(text: string) {
    const __args = { text };
    let __writer = new BinaryWriter(1024);
    SendMessage.getTypeScriptAlgebraicType().serialize(__writer, __args);
    let __argsBuffer = __writer.getBuffer();
    this.connection.callReducer("send_message", __argsBuffer, this.setCallReducerFlags.sendMessageFlags);
  }

  onSendMessage(callback: (ctx: ReducerEventContext, text: string) => void) {
    this.connection.onReducer("send_message", callback);
  }

  removeOnSendMessage(callback: (ctx: ReducerEventContext, text: string) => void) {
    this.connection.offReducer("send_message", callback);
  }

  setName(name: string) {
    const __args = { name };
    let __writer = new BinaryWriter(1024);
    SetName.getTypeScriptAlgebraicType().serialize(__writer, __args);
    let __argsBuffer = __writer.getBuffer();
    this.connection.callReducer("set_name", __argsBuffer, this.setCallReducerFlags.setNameFlags);
  }

  onSetName(callback: (ctx: ReducerEventContext, name: string) => void) {
    this.connection.onReducer("set_name", callback);
  }

  removeOnSetName(callback: (ctx: ReducerEventContext, name: string) => void) {
    this.connection.offReducer("set_name", callback);
  }

  startGame(lobbyId: bigint) {
    const __args = { lobbyId };
    let __writer = new BinaryWriter(1024);
    StartGame.getTypeScriptAlgebraicType().serialize(__writer, __args);
    let __argsBuffer = __writer.getBuffer();
    this.connection.callReducer("start_game", __argsBuffer, this.setCallReducerFlags.startGameFlags);
  }

  onStartGame(callback: (ctx: ReducerEventContext, lobbyId: bigint) => void) {
    this.connection.onReducer("start_game", callback);
  }

  removeOnStartGame(callback: (ctx: ReducerEventContext, lobbyId: bigint) => void) {
    this.connection.offReducer("start_game", callback);
  }

  takeCards(gameId: bigint, turnId: bigint) {
    const __args = { gameId, turnId };
    let __writer = new BinaryWriter(1024);
    TakeCards.getTypeScriptAlgebraicType().serialize(__writer, __args);
    let __argsBuffer = __writer.getBuffer();
    this.connection.callReducer("take_cards", __argsBuffer, this.setCallReducerFlags.takeCardsFlags);
  }

  onTakeCards(callback: (ctx: ReducerEventContext, gameId: bigint, turnId: bigint) => void) {
    this.connection.onReducer("take_cards", callback);
  }

  removeOnTakeCards(callback: (ctx: ReducerEventContext, gameId: bigint, turnId: bigint) => void) {
    this.connection.offReducer("take_cards", callback);
  }

  updateGameSettings(lobbyId: bigint, deckSize: DeckSize, startingCards: number, maxAttackCards: number, multiRoundMode: boolean, maxPoints: number, anyoneCanAttack: boolean, trumpCardToPlayer: boolean) {
    const __args = { lobbyId, deckSize, startingCards, maxAttackCards, multiRoundMode, maxPoints, anyoneCanAttack, trumpCardToPlayer };
    let __writer = new BinaryWriter(1024);
    UpdateGameSettings.getTypeScriptAlgebraicType().serialize(__writer, __args);
    let __argsBuffer = __writer.getBuffer();
    this.connection.callReducer("update_game_settings", __argsBuffer, this.setCallReducerFlags.updateGameSettingsFlags);
  }

  onUpdateGameSettings(callback: (ctx: ReducerEventContext, lobbyId: bigint, deckSize: DeckSize, startingCards: number, maxAttackCards: number, multiRoundMode: boolean, maxPoints: number, anyoneCanAttack: boolean, trumpCardToPlayer: boolean) => void) {
    this.connection.onReducer("update_game_settings", callback);
  }

  removeOnUpdateGameSettings(callback: (ctx: ReducerEventContext, lobbyId: bigint, deckSize: DeckSize, startingCards: number, maxAttackCards: number, multiRoundMode: boolean, maxPoints: number, anyoneCanAttack: boolean, trumpCardToPlayer: boolean) => void) {
    this.connection.offReducer("update_game_settings", callback);
  }

}

export class SetReducerFlags {
  attackFlags: CallReducerFlags = 'FullUpdate';
  attack(flags: CallReducerFlags) {
    this.attackFlags = flags;
  }

  createLobbyFlags: CallReducerFlags = 'FullUpdate';
  createLobby(flags: CallReducerFlags) {
    this.createLobbyFlags = flags;
  }

  defendFlags: CallReducerFlags = 'FullUpdate';
  defend(flags: CallReducerFlags) {
    this.defendFlags = flags;
  }

  joinLobbyFlags: CallReducerFlags = 'FullUpdate';
  joinLobby(flags: CallReducerFlags) {
    this.joinLobbyFlags = flags;
  }

  leaveLobbyFlags: CallReducerFlags = 'FullUpdate';
  leaveLobby(flags: CallReducerFlags) {
    this.leaveLobbyFlags = flags;
  }

  passTurnFlags: CallReducerFlags = 'FullUpdate';
  passTurn(flags: CallReducerFlags) {
    this.passTurnFlags = flags;
  }

  sendMessageFlags: CallReducerFlags = 'FullUpdate';
  sendMessage(flags: CallReducerFlags) {
    this.sendMessageFlags = flags;
  }

  setNameFlags: CallReducerFlags = 'FullUpdate';
  setName(flags: CallReducerFlags) {
    this.setNameFlags = flags;
  }

  startGameFlags: CallReducerFlags = 'FullUpdate';
  startGame(flags: CallReducerFlags) {
    this.startGameFlags = flags;
  }

  takeCardsFlags: CallReducerFlags = 'FullUpdate';
  takeCards(flags: CallReducerFlags) {
    this.takeCardsFlags = flags;
  }

  updateGameSettingsFlags: CallReducerFlags = 'FullUpdate';
  updateGameSettings(flags: CallReducerFlags) {
    this.updateGameSettingsFlags = flags;
  }

}

export class RemoteTables {
  constructor(private connection: DbConnectionImpl) {}

  get draw(): DrawTableHandle {
    return new DrawTableHandle(this.connection.clientCache.getOrCreateTable<Draw>(REMOTE_MODULE.tables.draw));
  }

  get game(): GameTableHandle {
    return new GameTableHandle(this.connection.clientCache.getOrCreateTable<Game>(REMOTE_MODULE.tables.game));
  }

  get gameSettings(): GameSettingsTableHandle {
    return new GameSettingsTableHandle(this.connection.clientCache.getOrCreateTable<GameSettings>(REMOTE_MODULE.tables.game_settings));
  }

  get lobby(): LobbyTableHandle {
    return new LobbyTableHandle(this.connection.clientCache.getOrCreateTable<Lobby>(REMOTE_MODULE.tables.lobby));
  }

  get message(): MessageTableHandle {
    return new MessageTableHandle(this.connection.clientCache.getOrCreateTable<Message>(REMOTE_MODULE.tables.message));
  }

  get playerCard(): PlayerCardTableHandle {
    return new PlayerCardTableHandle(this.connection.clientCache.getOrCreateTable<PlayerCard>(REMOTE_MODULE.tables.player_card));
  }

  get round(): RoundTableHandle {
    return new RoundTableHandle(this.connection.clientCache.getOrCreateTable<Round>(REMOTE_MODULE.tables.round));
  }

  get turn(): TurnTableHandle {
    return new TurnTableHandle(this.connection.clientCache.getOrCreateTable<Turn>(REMOTE_MODULE.tables.turn));
  }

  get user(): UserTableHandle {
    return new UserTableHandle(this.connection.clientCache.getOrCreateTable<User>(REMOTE_MODULE.tables.user));
  }
}

export class SubscriptionBuilder extends SubscriptionBuilderImpl<RemoteTables, RemoteReducers, SetReducerFlags> { }

export class DbConnection extends DbConnectionImpl<RemoteTables, RemoteReducers, SetReducerFlags> {
  static builder = (): DbConnectionBuilder<DbConnection, ErrorContext, SubscriptionEventContext> => {
    return new DbConnectionBuilder<DbConnection, ErrorContext, SubscriptionEventContext>(REMOTE_MODULE, (imp: DbConnectionImpl) => imp as DbConnection);
  }
  subscriptionBuilder = (): SubscriptionBuilder => {
    return new SubscriptionBuilder(this);
  }
}

export type EventContext = EventContextInterface<RemoteTables, RemoteReducers, SetReducerFlags, Reducer>;
export type ReducerEventContext = ReducerEventContextInterface<RemoteTables, RemoteReducers, SetReducerFlags, Reducer>;
export type SubscriptionEventContext = SubscriptionEventContextInterface<RemoteTables, RemoteReducers, SetReducerFlags>;
export type ErrorContext = ErrorContextInterface<RemoteTables, RemoteReducers, SetReducerFlags>;

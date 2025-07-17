# Durak (Fool) Card Game - Backend Design

## Overview
Durak is a traditional Russian card game where the goal is to get rid of all your cards. The last player with cards becomes the "Fool".

## Terms and Definitions

### Game Structure
- **Durak** (Дурак) - "Fool" in Russian; the overall loser of the game who reaches the point limit first
- **Game** - Complete session from lobby creation to final winner determination
- **Hand/Round** - Single dealing of cards that ends when only one player has cards remaining
- **Turn** - Period where one player defends against attacks from others
- **Draw** - Single card interaction (attack card + optional defense card)

### Gameplay Terms
- **Kozyr** (Козырь) - Trump suit; the suit that beats all other suits
- **Attack** - Playing a card against the defender
- **Defense** - Playing a card to beat an attacking card
- **Attacker** - Player who initiates or joins an attack
- **Defender** - Player who must beat all attacking cards or take them
- **Beat/Cover** - Successfully defend against an attack with a higher card
- **Take** - When defender cannot or chooses not to beat cards; takes all cards on table

### Card Terms
- **Deck** - Remaining undealt cards (колода)
- **Hand** - Cards a player holds (рука)
- **Table** - Area where attacking and defending cards are played
- **Discard** - Cards that have been beaten and removed from play
- **Trump Card** - The card that reveals the trump suit (traditionally goes to last dealt player)

### Player States
- **Active** - Player currently in the game
- **Finished** - Player who emptied their hand and left the current round
- **Left** - Player who quit the game early (receives penalty points)
- **Online/Offline** - Connection status

### Scoring Terms
- **Points** - Penalty points accumulated across hands (lower is better)
- **Loser Points** - 5 points given to the last player with cards in a hand
- **Early Leave Points** - 1 point given to players who quit mid-game
- **Winner Points** - 0 points for first player to empty their hand

### Technical Terms
- **Lobby** - Waiting room where players gather before starting a game
- **Settings** - Configurable game rules (attack limits, deck size, etc.)
- **Turn Order** - Clockwise sequence of players for determining next attacker
- **Refill** - Drawing cards from deck to maintain hand size (typically 7)

### Card Values and Suits
- **Ranks** - 6, 7, 8, 9, 10, Jack (Валет), Queen (Дама), King (Король), Ace (Туз)
- **Suits** - Hearts (Червы), Diamonds (Бубны), Clubs (Трефы), Spades (Пики)
- **Higher/Lower** - Card ranking where 6 is lowest, Ace is highest (except trump beats non-trump)

### Game Flow Terms
- **Deal** - Initial distribution of cards to players
- **Starting Player** - Player with lowest trump card (or lowest overall) who attacks first
- **Pass Turn** - When attacker cannot or chooses not to add more cards
- **End Hand** - When defender takes cards or all attacks are beaten
- **Shuffle** - Randomizing deck before dealing
- **Trump Reveal** - Showing bottom card to determine trump suit

## Game Rules Summary

### Setup
- 2-6 players (minimum 2)
- 36-card deck (6, 7, 8, 9, 10, J, Q, K, A in each suit)
- Each player starts with 7 cards (configurable)
- Trump suit (kozyr) is determined by the bottom card of the deck
- This trump card can go to a player at the end of dealing
- Player with the lowest trump card starts (or lowest overall card if no trumps on hand)

### Gameplay Flow
1. **Attack Phase**: Attacker plays a card against a defender
2. **Defense Phase**: Defender must beat the card with:
   - Higher rank of the same suit, OR
   - Any trump card (if attacking card isn't trump), OR
   - Higher trump card (if attacking card is trump)
3. **Additional Attacks**: Other players can join attack with cards matching any rank on the table (configurable)
4. **Attack Limits**: Maximum cards that can be played against defender (default 6, 0 = no limit)
5. **Resolution**: If defender beats all cards, they're discarded. If not, defender takes all cards
6. **Refill**: Players draw cards to maintain hand size (attackers first, then defender)
7. **Next Turn**: 
   - If defender succeeded: defender becomes next attacker
   - If defender failed: next player clockwise becomes attacker (defender skips turn)

### Winning Conditions
- Each hand continues until only one player has cards (everyone else is empty)
- Last player with cards loses the hand
- **Multi-Round Mode** (default): Traditional Durak with point system where loser gets 5 points, early leavers get 1 point, first to leave gets 0. First player to reach point limit becomes the "Fool" (overall loser)
- **Single Round Mode** (configurable): Game ends after one hand, loser is determined immediately

## Data Models

### Core Entities

#### User (consolidated entity)
```rust
pub struct User {
    #[primary_key]
    identity: Identity,
    name: Option<String>,
    online: bool,
    
    // Lobby state (if in a lobby)
    current_lobby_id: Option<u64>,
    lobby_joined_at: Option<Timestamp>,
    
    // Game state (if in a game)
    current_game_id: Option<u64>,
    game_position: Option<u8>, // 0-5, determines turn order
    total_points: Option<u8>, // Points accumulated across hands
    player_status: Option<PlayerStatus>, // Active, Left, Finished
}
```

#### Lobby
```rust
pub struct Lobby {
    #[primary_key]
    id: u64,
    name: String,
    creator: Identity,
    max_players: u8,
    current_players: u8,
    status: LobbyStatus, // Waiting, InGame, Finished
    created_at: Timestamp,
}
```

#### Game
```rust
pub struct Game {
    #[primary_key]
    id: u64,
    lobby_id: u64,
    status: GameStatus, // Active, Finished
    trump_suit: Suit,
    current_round: u32,
    started_at: Timestamp,
    finished_at: Option<Timestamp>,
}
```

#### Round
```rust
pub struct Round {
    #[primary_key]
    id: u64,
    game_id: u64,
    round_number: u32,
    status: RoundStatus, // Active, Finished
    loser: Option<Identity>, // Who lost this hand/round
    started_at: Timestamp,
    finished_at: Option<Timestamp>,
}
```

#### Turn
```rust
pub struct Turn {
    #[primary_key]
    id: u64,
    round_id: u64,
    turn_number: u32,
    attacker: Identity,
    defender: Identity,
    status: TurnStatus, // Active, DefenderTook, DefenderBeat
    started_at: Timestamp,
    finished_at: Option<Timestamp>,
}
```

#### Draw
```rust
pub struct Draw {
    #[primary_key]
    id: u64,
    turn_id: u64,
    attacker: Identity,
    attacking_card: Card,
    defending_card: Option<Card>,
    status: DrawStatus, // Pending, Beaten, Taken
    created_at: Timestamp,
}
```

#### Card & Player Cards
```rust
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Card {
    suit: Suit, // Hearts, Diamonds, Clubs, Spades
    rank: Rank, // Six=6, Seven=7, ..., Ace=14
}

#[table(name = player_card, public)]
pub struct PlayerCard {
    #[primary_key]
    id: u64,
    game_id: u64,
    player: Identity,
    card: Card,
    location: CardLocation, // Hand, Deck, Discarded, OnTable
}
```

#### GameSettings
```rust
pub struct GameSettings {
    #[primary_key]
    lobby_id: u64,
    deck_size: DeckSize, // Standard36 (default), Extended52
    starting_cards: u8, // Default 7 (traditional)
    max_attack_cards: u8, // Default 6 (traditional limit), 0 = no limit
    multi_round_mode: bool, // Default true (traditional Durak)
    max_points: u8, // Default 15 (traditional "Fool" threshold)
    anyone_can_attack: bool, // Default true (traditional - any player can join attack)
    trump_card_to_player: bool, // Default true (traditional - trump card goes to last dealt player)
}
```

### Enums
```rust
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Suit {
    Hearts,   // Червы
    Diamonds, // Бубны  
    Clubs,    // Трефы
    Spades,   // Пики
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]
pub enum Rank {
    Six = 6,
    Seven = 7,
    Eight = 8,
    Nine = 9,
    Ten = 10,
    Jack = 11,
    Queen = 12,
    King = 13,
    Ace = 14,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum LobbyStatus {
    Waiting,
    InGame,
    Finished,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum GameStatus {
    Active,
    Finished,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PlayerStatus {
    Active,
    Left,      // Quit early
    Finished,  // Emptied hand successfully
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum RoundStatus {
    Active,
    Finished,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum TurnStatus {
    Active,
    DefenderTook,  // Defender took cards
    DefenderBeat,  // Defender beat all attacks
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DrawStatus {
    Pending,  // Attack card played, waiting for defense
    Beaten,   // Successfully defended
    Taken,    // Defender took this card
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum CardLocation {
    Hand,
    Deck,
    Discarded,
    OnTable,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DeckSize {
    Standard36,  // Traditional 6-A
    Extended52,  // Full deck 2-A
}
```

## Data Model Benefits

**Simplified Architecture**: Single User entity eliminates joins and complex relationships  
**SpacetimeDB Logging**: Automatic state change tracking replaces manual history tables  
**Atomic Updates**: User state changes are atomic and consistent  
**Easier Queries**: No need to join across User/LobbyMember/Player tables  
**Better Performance**: Fewer tables = faster queries and updates

### Query Examples
```rust
// Get all players in a lobby (before: join User + LobbyMember)
ctx.db.user().current_lobby_id().filter(lobby_id);

// Get all players in a game (before: join User + Player)  
ctx.db.user().current_game_id().filter(game_id);

// Update user from lobby to game (atomic operation)
ctx.db.user().identity().update(User {
    current_lobby_id: None,
    current_game_id: Some(game_id),
    game_position: Some(position),
    total_points: Some(0),
    player_status: Some(PlayerStatus::Active),
    ..user
});
```

## API (Reducers)

### Lobby Management
- `create_lobby(name: String, max_players: u8)` 
- `join_lobby(lobby_id: u64)`
- `leave_lobby(lobby_id: u64)`
- `update_game_settings(lobby_id: u64, settings: GameSettings)` // Only lobby creator
- `start_game(lobby_id: u64)` // Only lobby creator
- `list_lobbies()` (query)

### Game Actions
- `attack(game_id: u64, card: Card, target: Identity)`
- `defend(game_id: u64, turn_id: u64, card: Card)`
- `take_cards(game_id: u64, turn_id: u64)`
- `pass_turn(game_id: u64)` // When no more attacks possible

### Game Queries
- `get_game_state(game_id: u64)` - Full game state with all players
- `get_player_hand(game_id: u64)` - Current player's cards
- `get_current_turn(game_id: u64)` - Active turn info
- `get_lobby_players(lobby_id: u64)` - All users in a lobby
- `get_game_players(game_id: u64)` - All users in a game

## Game State Management

### Card Dealing
- Shuffle 36-card deck (6, 7, 8, 9, 10, J, Q, K, A in each suit)
- Deal starting number of cards to each player (default 7, traditional)
- Set trump suit (kozyr) from bottom card of remaining deck
- Traditionally, trump card goes to the last player dealt (configurable)
- Determine starting player (lowest trump card in hand, or lowest overall if no trumps in any hand)

### Turn Resolution
1. Validate attack is legal
2. Check if defense is valid
3. Allow additional attacks with matching ranks
4. Resolve turn (cards taken or beaten)
5. Refill hands
6. Determine next attacker

### Round/Game End Conditions
- **Hand End**: Each hand ends when only one player has cards remaining
- **Traditional Mode**: Game continues through multiple hands until someone reaches point limit (15) and becomes the "Fool"
- **Single Hand Mode**: Game ends after one hand, immediate winner/loser
- **Scoring**: Handle point accumulation and determine overall winner
- **Player elimination**: Handle early departures with appropriate point penalties

## Technical Considerations

### SpacetimeDB Integration
- **Simplified Schema**: Single User entity reduces complexity
- **Automatic Logging**: SpacetimeDB tracks all state changes for history/debugging
- **Real-time Updates**: All clients receive instant updates when user state changes
- **Generated Bindings**: Client code auto-generated from Rust structs
- **Atomic Transactions**: User state updates are atomic (lobby + game state together)

### Validation
- **Defense validation**: Must use higher same suit, any trump (vs non-trump), or higher trump (vs trump)
- **Attack validation**: Can only attack with ranks already on the table (after initial attack)
- **Attack limits**: Respect max_attack_cards setting
- **Turn order**: Enforce who can attack based on settings (anyone vs clockwise only)
- **Hand management**: Ensure proper card dealing and refill order
- **User state consistency**: Validate user can only be in one lobby/game at a time
- **Game state consistency**: Validate all moves against current game state

### Error Handling
- Invalid moves
- Player disconnections
- Game state corruption recovery

## Future Enhancements
- Spectator mode
- Game replays
- Tournament brackets
- AI players
- Custom rule variations 
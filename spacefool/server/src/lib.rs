use spacetimedb::{table, reducer, Table, ReducerContext, Identity, Timestamp, SpacetimeType};

// Core game enums
#[derive(Debug, Clone, Copy, PartialEq, Eq, SpacetimeType)]
pub enum Suit {
    Hearts,   // Червы
    Diamonds, // Бубны  
    Clubs,    // Трефы
    Spades,   // Пики
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, SpacetimeType)]
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

#[derive(Debug, Clone, PartialEq, Eq, SpacetimeType)]
pub struct Card {
    suit: Suit,
    rank: Rank,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, SpacetimeType)]
pub enum LobbyStatus {
    Waiting,
    InGame,
    Finished,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, SpacetimeType)]
pub enum GameStatus {
    Active,
    Finished,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, SpacetimeType)]
pub enum PlayerStatus {
    Active,
    Left,      // Quit early
    Finished,  // Emptied hand successfully
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, SpacetimeType)]
pub enum RoundStatus {
    Active,
    Finished,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, SpacetimeType)]
pub enum TurnStatus {
    Active,
    DefenderTook,  // Defender took cards
    DefenderBeat,  // Defender beat all attacks
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, SpacetimeType)]
pub enum DrawStatus {
    Pending,  // Attack card played, waiting for defense
    Beaten,   // Successfully defended
    Taken,    // Defender took this card
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, SpacetimeType)]
pub enum CardLocation {
    Hand,
    Deck,
    Discarded,
    OnTable,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, SpacetimeType)]
pub enum DeckSize {
    Standard36,  // Traditional 6-A
    Extended52,  // Full deck 2-A
}

#[table(name = user, public)]
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

#[table(name = lobby, public)]
pub struct Lobby {
    #[primary_key]
    id: u64,
    name: String,
    creator: Identity,
    max_players: u8,
    current_players: u8,
    status: LobbyStatus,
    created_at: Timestamp,
}

#[table(name = game, public)]
pub struct Game {
    #[primary_key]
    id: u64,
    lobby_id: u64,
    status: GameStatus,
    trump_suit: Suit,
    current_round: u32,
    started_at: Timestamp,
    finished_at: Option<Timestamp>,
}

#[table(name = game_settings, public)]
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

#[derive(Clone)]
#[table(name = round, public)]
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

#[derive(Clone)]
#[table(name = turn, public)]
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

#[derive(Clone)]
#[table(name = draw, public)]
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

#[derive(Clone)]
#[table(name = player_card, public)]
pub struct PlayerCard {
    #[primary_key]
    id: u64,
    game_id: u64,
    player: Identity,
    card: Card,
    location: CardLocation, // Hand, Deck, Discarded, OnTable
}

#[table(name = message, public)]
pub struct Message {
    sender: Identity,
    sent: Timestamp,
    text: String,
}


#[reducer]
/// Clients invoke this reducer to set their user names.
pub fn set_name(ctx: &ReducerContext, name: String) -> Result<(), String> {
    let name = validate_name(name)?;
    if let Some(user) = ctx.db.user().identity().find(ctx.sender) {
        ctx.db.user().identity().update(User { name: Some(name), ..user });
        Ok(())
    } else {
        Err("Cannot set name for unknown user".to_string())
    }
}

/// Takes a name and checks if it's acceptable as a user's name.
fn validate_name(name: String) -> Result<String, String> {
    if name.is_empty() {
        Err("Names must not be empty".to_string())
    } else {
        Ok(name)
    }
}

#[reducer]
/// Clients invoke this reducer to send messages.
pub fn send_message(ctx: &ReducerContext, text: String) -> Result<(), String> {
    let text = validate_message(text)?;
    log::info!("{}", text);
    ctx.db.message().insert(Message {
        sender: ctx.sender,
        text,
        sent: ctx.timestamp,
    });
    Ok(())
}

/// Takes a message's text and checks if it's acceptable to send.
fn validate_message(text: String) -> Result<String, String> {
    if text.is_empty() {
        Err("Messages must not be empty".to_string())
    } else {
        Ok(text)
    }
}

#[reducer(client_connected)]
// Called when a client connects to a SpacetimeDB database server
pub fn client_connected(ctx: &ReducerContext) {
    if let Some(user) = ctx.db.user().identity().find(ctx.sender) {
        // If this is a returning user, i.e. we already have a `User` with this `Identity`,
        // set `online: true`, but leave other fields unchanged.
        ctx.db.user().identity().update(User { online: true, ..user });
    } else {
        // If this is a new user, create a `User` row for the `Identity`,
        // which is online, but hasn't set a name or joined any lobbies/games.
        ctx.db.user().insert(User {
            name: None,
            identity: ctx.sender,
            online: true,
            current_lobby_id: None,
            lobby_joined_at: None,
            current_game_id: None,
            game_position: None,
            total_points: None,
            player_status: None,
        });
    }
}

#[reducer(client_disconnected)]
// Called when a client disconnects from SpacetimeDB database server
pub fn identity_disconnected(ctx: &ReducerContext) {
    if let Some(user) = ctx.db.user().identity().find(ctx.sender) {
        ctx.db.user().identity().update(User { online: false, ..user });
    } else {
        // This branch should be unreachable,
        // as it doesn't make sense for a client to disconnect without connecting first.
        log::warn!("Disconnect event for unknown user with identity {:?}", ctx.sender);
    }
}

// Lobby Management

/// Generate a unique lobby ID (simple counter approach for now)
fn generate_lobby_id(_timestamp: Timestamp) -> u64 {
    // For now, use a simple random-like ID. In production, this could be more sophisticated.
    use std::collections::hash_map::DefaultHasher;
    use std::hash::{Hash, Hasher};
    
    let mut hasher = DefaultHasher::new();
    _timestamp.hash(&mut hasher);
    hasher.finish()
}

#[reducer]
/// Creates a new lobby with the specified name and max players
pub fn create_lobby(ctx: &ReducerContext, name: String, max_players: u8) -> Result<(), String> {
    if name.is_empty() {
        return Err("Lobby name cannot be empty".to_string());
    }
    
    if max_players < 2 || max_players > 6 {
        return Err("Max players must be between 2 and 6".to_string());
    }

    let user = ctx.db.user().identity().find(ctx.sender)
        .ok_or("User not found")?;

    if user.current_lobby_id.is_some() {
        return Err("You are already in a lobby".to_string());
    }

    if user.current_game_id.is_some() {
        return Err("You are currently in a game".to_string());
    }

    let lobby_id = generate_lobby_id(ctx.timestamp);
    
    // Create the lobby
    ctx.db.lobby().insert(Lobby {
        id: lobby_id,
        name,
        creator: ctx.sender,
        max_players,
        current_players: 1,
        status: LobbyStatus::Waiting,
        created_at: ctx.timestamp,
    });

    // Update user to join the lobby
    ctx.db.user().identity().update(User {
        current_lobby_id: Some(lobby_id),
        lobby_joined_at: Some(ctx.timestamp),
        ..user
    });

    log::info!("User {:?} created lobby {}", ctx.sender, lobby_id);
    Ok(())
}

#[reducer]
/// Join an existing lobby by ID
pub fn join_lobby(ctx: &ReducerContext, lobby_id: u64) -> Result<(), String> {
    let user = ctx.db.user().identity().find(ctx.sender)
        .ok_or("User not found")?;

    if user.current_lobby_id.is_some() {
        return Err("You are already in a lobby".to_string());
    }

    if user.current_game_id.is_some() {
        return Err("You are currently in a game".to_string());
    }

    let lobby = ctx.db.lobby().id().find(lobby_id)
        .ok_or("Lobby not found")?;

    if lobby.status != LobbyStatus::Waiting {
        return Err("Lobby is not accepting new players".to_string());
    }

    if lobby.current_players >= lobby.max_players {
        return Err("Lobby is full".to_string());
    }

    // Update lobby player count
    ctx.db.lobby().id().update(Lobby {
        current_players: lobby.current_players + 1,
        ..lobby
    });

    // Update user to join the lobby
    ctx.db.user().identity().update(User {
        current_lobby_id: Some(lobby_id),
        lobby_joined_at: Some(ctx.timestamp),
        ..user
    });

    log::info!("User {:?} joined lobby {}", ctx.sender, lobby_id);
    Ok(())
}

#[reducer]
/// Leave the current lobby
pub fn leave_lobby(ctx: &ReducerContext) -> Result<(), String> {
    let user = ctx.db.user().identity().find(ctx.sender)
        .ok_or("User not found")?;

    let lobby_id = user.current_lobby_id
        .ok_or("You are not in a lobby")?;

    let lobby = ctx.db.lobby().id().find(lobby_id)
        .ok_or("Lobby not found")?;

    // Update lobby player count
    let new_player_count = lobby.current_players.saturating_sub(1);
    
    if new_player_count == 0 || lobby.creator == ctx.sender {
        // If lobby is empty or creator left, delete the lobby
        ctx.db.lobby().id().delete(lobby_id);
        log::info!("Lobby {} deleted", lobby_id);
    } else {
        // Just update player count
        ctx.db.lobby().id().update(Lobby {
            current_players: new_player_count,
            ..lobby
        });
    }

    // Update user to leave the lobby
    ctx.db.user().identity().update(User {
        current_lobby_id: None,
        lobby_joined_at: None,
        ..user
    });

    log::info!("User {:?} left lobby {}", ctx.sender, lobby_id);
    Ok(())
}

// Game Settings Management

#[reducer]
/// Update game settings for a lobby (only creator can do this)
pub fn update_game_settings(
    ctx: &ReducerContext, 
    lobby_id: u64,
    deck_size: DeckSize,
    starting_cards: u8,
    max_attack_cards: u8,
    multi_round_mode: bool,
    max_points: u8,
    anyone_can_attack: bool,
    trump_card_to_player: bool
) -> Result<(), String> {
    let user = ctx.db.user().identity().find(ctx.sender)
        .ok_or("User not found")?;

    if user.current_lobby_id != Some(lobby_id) {
        return Err("You are not in this lobby".to_string());
    }

    let lobby = ctx.db.lobby().id().find(lobby_id)
        .ok_or("Lobby not found")?;

    if lobby.creator != ctx.sender {
        return Err("Only lobby creator can change settings".to_string());
    }

    if lobby.status != LobbyStatus::Waiting {
        return Err("Cannot change settings after game has started".to_string());
    }

    // Validate settings
    if starting_cards < 3 || starting_cards > 20 {
        return Err("Starting cards must be between 3 and 20".to_string());
    }

    if max_points < 5 || max_points > 50 {
        return Err("Max points must be between 5 and 50".to_string());
    }

    // Insert or update settings
    if let Some(existing) = ctx.db.game_settings().lobby_id().find(lobby_id) {
        ctx.db.game_settings().lobby_id().update(GameSettings {
            deck_size,
            starting_cards,
            max_attack_cards,
            multi_round_mode,
            max_points,
            anyone_can_attack,
            trump_card_to_player,
            ..existing
        });
    } else {
        ctx.db.game_settings().insert(GameSettings {
            lobby_id,
            deck_size,
            starting_cards,
            max_attack_cards,
            multi_round_mode,
            max_points,
            anyone_can_attack,
            trump_card_to_player,
        });
    }

    log::info!("Game settings updated for lobby {}", lobby_id);
    Ok(())
}

/// Get default game settings
fn get_default_settings(lobby_id: u64) -> GameSettings {
    GameSettings {
        lobby_id,
        deck_size: DeckSize::Standard36,
        starting_cards: 7,
        max_attack_cards: 6,
        multi_round_mode: true,
        max_points: 15,
        anyone_can_attack: true,
        trump_card_to_player: true,
    }
}

// Card and Deck Management

/// Generate a full deck based on deck size setting
fn create_deck(deck_size: DeckSize) -> Vec<Card> {
    let mut deck = Vec::new();
    let suits = [Suit::Hearts, Suit::Diamonds, Suit::Clubs, Suit::Spades];
    
    let ranks = match deck_size {
        DeckSize::Standard36 => vec![
            Rank::Six, Rank::Seven, Rank::Eight, Rank::Nine, Rank::Ten,
            Rank::Jack, Rank::Queen, Rank::King, Rank::Ace
        ],
        DeckSize::Extended52 => vec![
            Rank::Six, Rank::Seven, Rank::Eight, Rank::Nine, Rank::Ten,
            Rank::Jack, Rank::Queen, Rank::King, Rank::Ace
        ], // TODO: Add ranks 2-5 for extended deck
    };

    for suit in suits {
        for rank in &ranks {
            deck.push(Card { suit, rank: *rank });
        }
    }

    deck
}

/// Shuffle deck using timestamp-based seeding
fn shuffle_deck(mut deck: Vec<Card>, timestamp: Timestamp) -> Vec<Card> {
    use std::collections::hash_map::DefaultHasher;
    use std::hash::{Hash, Hasher};
    
    // Create a deterministic but unpredictable seed
    let mut hasher = DefaultHasher::new();
    timestamp.hash(&mut hasher);
    let seed = hasher.finish();
    
    // Simple Fisher-Yates shuffle with our seed
    for i in (1..deck.len()).rev() {
        let j = (seed.wrapping_mul(i as u64 + 1) % (i as u64 + 1)) as usize;
        deck.swap(i, j);
    }
    
    deck
}

/// Generate unique IDs for game entities
fn generate_game_id(timestamp: Timestamp) -> u64 {
    use std::collections::hash_map::DefaultHasher;
    use std::hash::{Hash, Hasher};
    
    let mut hasher = DefaultHasher::new();
    timestamp.hash(&mut hasher);
    hasher.finish()
}

fn generate_round_id(game_id: u64, round_number: u32) -> u64 {
    use std::collections::hash_map::DefaultHasher;
    use std::hash::{Hash, Hasher};
    
    let mut hasher = DefaultHasher::new();
    game_id.hash(&mut hasher);
    round_number.hash(&mut hasher);
    hasher.finish()
}

#[reducer]
/// Start the game from a lobby (only creator can do this)
pub fn start_game(ctx: &ReducerContext, lobby_id: u64) -> Result<(), String> {
    let user = ctx.db.user().identity().find(ctx.sender)
        .ok_or("User not found")?;

    if user.current_lobby_id != Some(lobby_id) {
        return Err("You are not in this lobby".to_string());
    }

    let lobby = ctx.db.lobby().id().find(lobby_id)
        .ok_or("Lobby not found")?;

    if lobby.creator != ctx.sender {
        return Err("Only lobby creator can start the game".to_string());
    }

    if lobby.status != LobbyStatus::Waiting {
        return Err("Game has already been started".to_string());
    }

    if lobby.current_players < 2 {
        return Err("Need at least 2 players to start".to_string());
    }

    // Get or create game settings
    let settings = ctx.db.game_settings().lobby_id().find(lobby_id)
        .unwrap_or_else(|| get_default_settings(lobby_id));

    // Get all players in the lobby
    let players: Vec<User> = ctx.db.user()
        .iter()
        .filter(|user| user.current_lobby_id == Some(lobby_id))
        .collect();

    if players.len() != lobby.current_players as usize {
        return Err("Player count mismatch".to_string());
    }

    // Generate deck and determine trump suit
    let deck = create_deck(settings.deck_size);
    let shuffled_deck = shuffle_deck(deck, ctx.timestamp);
    
    // Trump suit is the suit of the last card (bottom of deck)
    let trump_suit = shuffled_deck.last().unwrap().suit;

    // Create game
    let game_id = generate_game_id(ctx.timestamp);
    ctx.db.game().insert(Game {
        id: game_id,
        lobby_id,
        status: GameStatus::Active,
        trump_suit,
        current_round: 1,
        started_at: ctx.timestamp,
        finished_at: None,
    });

    // Deal cards to players
    let mut card_index = 0;
    let mut card_id_counter = 0;

    // Deal starting cards to each player
    for (position, player) in players.iter().enumerate() {
        for _ in 0..settings.starting_cards {
            if card_index >= shuffled_deck.len() {
                return Err("Not enough cards in deck".to_string());
            }

            ctx.db.player_card().insert(PlayerCard {
                id: card_id_counter,
                game_id,
                player: player.identity,
                card: shuffled_deck[card_index].clone(),
                location: CardLocation::Hand,
            });

            card_index += 1;
            card_id_counter += 1;
        }

        // Update user to join game
        ctx.db.user().identity().update(User {
            identity: player.identity,
            name: player.name.clone(),
            online: player.online,
            current_lobby_id: None,
            lobby_joined_at: None,
            current_game_id: Some(game_id),
            game_position: Some(position as u8),
            total_points: Some(0),
            player_status: Some(PlayerStatus::Active),
        });
    }

    // Put remaining cards in deck
    for i in card_index..shuffled_deck.len() {
        ctx.db.player_card().insert(PlayerCard {
            id: card_id_counter,
            game_id,
            player: players[0].identity, // Assign to first player for now, doesn't matter for deck cards
            card: shuffled_deck[i].clone(),
            location: CardLocation::Deck,
        });
        card_id_counter += 1;
    }

    // If trump card goes to player (traditional rule)
    if settings.trump_card_to_player && !shuffled_deck.is_empty() {
        let trump_card = shuffled_deck.last().unwrap();
        // Find the trump card in deck and move to last player's hand
        let last_player = &players[players.len() - 1];
        
        // This is simplified - in real implementation you'd find the actual trump card record
        ctx.db.player_card().insert(PlayerCard {
            id: card_id_counter,
            game_id,
            player: last_player.identity,
            card: trump_card.clone(),
            location: CardLocation::Hand,
        });
    }

    // Create first round
    let round_id = generate_round_id(game_id, 1);
    ctx.db.round().insert(Round {
        id: round_id,
        game_id,
        round_number: 1,
        status: RoundStatus::Active,
        loser: None,
        started_at: ctx.timestamp,
        finished_at: None,
    });

    // Update lobby status
    ctx.db.lobby().id().update(Lobby {
        status: LobbyStatus::InGame,
        ..lobby
    });

    log::info!("Game {} started from lobby {} with {} players", game_id, lobby_id, players.len());
    Ok(())
}

// Query functions (these don't modify state, just return data)

/// Get all available lobbies that can be joined
pub fn get_available_lobbies(ctx: &ReducerContext) -> Vec<Lobby> {
    ctx.db.lobby()
        .iter()
        .filter(|lobby| lobby.status == LobbyStatus::Waiting)
        .collect()
}

/// Get all players in a specific lobby
pub fn get_lobby_players(ctx: &ReducerContext, lobby_id: u64) -> Vec<User> {
    ctx.db.user()
        .iter()
        .filter(|user| user.current_lobby_id == Some(lobby_id))
        .collect()
}

/// Get all players in a specific game
pub fn get_game_players(ctx: &ReducerContext, game_id: u64) -> Vec<User> {
    ctx.db.user()
        .iter()
        .filter(|user| user.current_game_id == Some(game_id))
        .collect()
}

/// Get current player's hand
pub fn get_player_hand(ctx: &ReducerContext, game_id: u64) -> Vec<Card> {
    ctx.db.player_card()
        .iter()
        .filter(|pc| pc.game_id == game_id && pc.player == ctx.sender && pc.location == CardLocation::Hand)
        .map(|pc| pc.card.clone())
        .collect()
}

/// Get current game state
pub fn get_game_state(ctx: &ReducerContext, game_id: u64) -> Option<Game> {
    ctx.db.game().id().find(game_id)
}

/// Get game settings for a lobby
pub fn get_game_settings(ctx: &ReducerContext, lobby_id: u64) -> GameSettings {
    ctx.db.game_settings()
        .lobby_id()
        .find(lobby_id)
        .unwrap_or_else(|| get_default_settings(lobby_id))
}

/// Get current round for a game
pub fn get_current_round(ctx: &ReducerContext, game_id: u64) -> Option<Round> {
    ctx.db.round()
        .iter()
        .filter(|round| round.game_id == game_id && round.status == RoundStatus::Active)
        .next()
}

// Card Validation Helpers

/// Check if a defending card can beat an attacking card
fn can_beat_card(attacking_card: &Card, defending_card: &Card, trump_suit: Suit) -> bool {
    let attack_is_trump = attacking_card.suit == trump_suit;
    let defend_is_trump = defending_card.suit == trump_suit;

    match (attack_is_trump, defend_is_trump) {
        // Trump vs trump: higher rank wins
        (true, true) => defending_card.rank > attacking_card.rank,
        // Non-trump vs trump: trump always wins
        (false, true) => true,
        // Trump vs non-trump: trump always wins (defense invalid)
        (true, false) => false,
        // Non-trump vs non-trump: same suit and higher rank
        (false, false) => {
            defending_card.suit == attacking_card.suit && defending_card.rank > attacking_card.rank
        }
    }
}

/// Check if an attacking card rank is valid (must match existing ranks on table)
fn is_valid_attack_rank(rank: Rank, turn_id: u64, ctx: &ReducerContext) -> bool {
    let existing_draws: Vec<Draw> = ctx.db.draw()
        .iter()
        .filter(|draw| draw.turn_id == turn_id)
        .collect();

    if existing_draws.is_empty() {
        // First attack can be any rank
        return true;
    }

    // Additional attacks must match existing ranks on table
    existing_draws.iter().any(|draw| {
        draw.attacking_card.rank == rank || 
        draw.defending_card.as_ref().map_or(false, |card| card.rank == rank)
    })
}

/// Get player's cards in hand
fn get_player_cards(ctx: &ReducerContext, game_id: u64, player: Identity) -> Vec<PlayerCard> {
    ctx.db.player_card()
        .iter()
        .filter(|pc| pc.game_id == game_id && pc.player == player && pc.location == CardLocation::Hand)
        .collect()
}

/// Check if player has the specified card in hand
fn player_has_card(ctx: &ReducerContext, game_id: u64, player: Identity, card: &Card) -> bool {
    get_player_cards(ctx, game_id, player)
        .iter()
        .any(|pc| pc.card == *card)
}

/// Generate unique turn ID
fn generate_turn_id(round_id: u64, turn_number: u32) -> u64 {
    use std::collections::hash_map::DefaultHasher;
    use std::hash::{Hash, Hasher};
    
    let mut hasher = DefaultHasher::new();
    round_id.hash(&mut hasher);
    turn_number.hash(&mut hasher);
    hasher.finish()
}

/// Generate unique draw ID
fn generate_draw_id(turn_id: u64, timestamp: Timestamp) -> u64 {
    use std::collections::hash_map::DefaultHasher;
    use std::hash::{Hash, Hasher};
    
    let mut hasher = DefaultHasher::new();
    turn_id.hash(&mut hasher);
    timestamp.hash(&mut hasher);
    hasher.finish()
}

/// Get current active turn for a round
fn get_active_turn(ctx: &ReducerContext, round_id: u64) -> Option<Turn> {
    ctx.db.turn()
        .iter()
        .filter(|turn| turn.round_id == round_id && turn.status == TurnStatus::Active)
        .next()
}

/// Count pending draws (attacks waiting for defense)
fn count_pending_draws(ctx: &ReducerContext, turn_id: u64) -> usize {
    ctx.db.draw()
        .iter()
        .filter(|draw| draw.turn_id == turn_id && draw.status == DrawStatus::Pending)
        .count()
}

/// Get game settings with defaults if not found
fn get_game_settings_for_game(ctx: &ReducerContext, game_id: u64) -> Result<GameSettings, String> {
    let game = ctx.db.game().id().find(game_id)
        .ok_or("Game not found")?;
    
    Ok(ctx.db.game_settings()
        .lobby_id()
        .find(game.lobby_id)
        .unwrap_or_else(|| get_default_settings(game.lobby_id)))
}

// Core Game Actions

#[reducer]
/// Attack another player with a card
pub fn attack(ctx: &ReducerContext, game_id: u64, card: Card, target: Identity) -> Result<(), String> {
    // Validate game exists and is active
    let game = ctx.db.game().id().find(game_id)
        .ok_or("Game not found")?;
    
    if game.status != GameStatus::Active {
        return Err("Game is not active".to_string());
    }

    // Validate attacker is in the game
    let attacker = ctx.db.user().identity().find(ctx.sender)
        .ok_or("User not found")?;
    
    if attacker.current_game_id != Some(game_id) {
        return Err("You are not in this game".to_string());
    }

    if attacker.player_status != Some(PlayerStatus::Active) {
        return Err("You are not active in this game".to_string());
    }

    // Validate target is in the game
    let defender = ctx.db.user().identity().find(target)
        .ok_or("Target player not found")?;
    
    if defender.current_game_id != Some(game_id) {
        return Err("Target player is not in this game".to_string());
    }

    if defender.player_status != Some(PlayerStatus::Active) {
        return Err("Target player is not active".to_string());
    }

    // Get current round
    let round = get_current_round(ctx, game_id)
        .ok_or("No active round found")?;

    // Check if attacker has the card
    if !player_has_card(ctx, game_id, ctx.sender, &card) {
        return Err("You don't have this card".to_string());
    }

    // Get current turn or create new one
    let turn = if let Some(existing_turn) = get_active_turn(ctx, round.id) {
        // Validate this is an additional attack on existing turn
        if existing_turn.defender != target {
            return Err("Can only attack the current defender".to_string());
        }

        // Check if rank is valid for additional attack
        if !is_valid_attack_rank(card.rank, existing_turn.id, ctx) {
            return Err("Attack card rank must match existing cards on table".to_string());
        }

        // Check attack limits
        let settings = get_game_settings_for_game(ctx, game_id)?;
        if settings.max_attack_cards > 0 {
            let current_attacks = ctx.db.draw()
                .iter()
                .filter(|draw| draw.turn_id == existing_turn.id)
                .count();
            
            if current_attacks >= settings.max_attack_cards as usize {
                return Err("Maximum attack cards reached".to_string());
            }
        }

        // Check if anyone can attack or just specific players
        if !settings.anyone_can_attack {
            // In traditional rules, only the original attacker can add cards
            if existing_turn.attacker != ctx.sender {
                return Err("Only the original attacker can add more cards".to_string());
            }
        }

        existing_turn
    } else {
        // Create new turn with this attack
        let turn_number = ctx.db.turn()
            .iter()
            .filter(|t| t.round_id == round.id)
            .count() as u32 + 1;

        let turn_id = generate_turn_id(round.id, turn_number);
        let new_turn = Turn {
            id: turn_id,
            round_id: round.id,
            turn_number,
            attacker: ctx.sender,
            defender: target,
            status: TurnStatus::Active,
            started_at: ctx.timestamp,
            finished_at: None,
        };

        ctx.db.turn().insert(new_turn.clone());
        new_turn
    };

    // Create the draw (attack)
    let draw_id = generate_draw_id(turn.id, ctx.timestamp);
    ctx.db.draw().insert(Draw {
        id: draw_id,
        turn_id: turn.id,
        attacker: ctx.sender,
        attacking_card: card.clone(),
        defending_card: None,
        status: DrawStatus::Pending,
        created_at: ctx.timestamp,
    });

    // Move card from hand to table
    if let Some(player_card) = ctx.db.player_card()
        .iter()
        .find(|pc| pc.game_id == game_id && pc.player == ctx.sender && 
                   pc.location == CardLocation::Hand && pc.card == card) {
        ctx.db.player_card().id().update(PlayerCard {
            location: CardLocation::OnTable,
            ..player_card
        });
    }

    log::info!("Player {:?} attacked {:?} with {:?} of {:?}", 
               ctx.sender, target, card.rank, card.suit);
    Ok(())
}

#[reducer]
/// Defend against an attack with a card
pub fn defend(ctx: &ReducerContext, game_id: u64, turn_id: u64, card: Card) -> Result<(), String> {
    // Validate game exists and is active
    let game = ctx.db.game().id().find(game_id)
        .ok_or("Game not found")?;
    
    if game.status != GameStatus::Active {
        return Err("Game is not active".to_string());
    }

    // Validate defender is in the game
    let defender = ctx.db.user().identity().find(ctx.sender)
        .ok_or("User not found")?;
    
    if defender.current_game_id != Some(game_id) {
        return Err("You are not in this game".to_string());
    }

    // Get the turn
    let turn = ctx.db.turn().id().find(turn_id)
        .ok_or("Turn not found")?;
    
    if turn.defender != ctx.sender {
        return Err("You are not the defender for this turn".to_string());
    }

    if turn.status != TurnStatus::Active {
        return Err("Turn is not active".to_string());
    }

    // Check if defender has the card
    if !player_has_card(ctx, game_id, ctx.sender, &card) {
        return Err("You don't have this card".to_string());
    }

    // Find a pending draw to defend against
    let pending_draw = ctx.db.draw()
        .iter()
        .find(|draw| draw.turn_id == turn_id && draw.status == DrawStatus::Pending)
        .ok_or("No attack to defend against")?;

    // Validate defense is legal
    if !can_beat_card(&pending_draw.attacking_card, &card, game.trump_suit) {
        return Err("Your card cannot beat the attacking card".to_string());
    }

    // Update the draw with defense
    ctx.db.draw().id().update(Draw {
        defending_card: Some(card.clone()),
        status: DrawStatus::Beaten,
        ..pending_draw
    });

    // Move defending card from hand to table
    if let Some(player_card) = ctx.db.player_card()
        .iter()
        .find(|pc| pc.game_id == game_id && pc.player == ctx.sender && 
                   pc.location == CardLocation::Hand && pc.card == card) {
        ctx.db.player_card().id().update(PlayerCard {
            location: CardLocation::OnTable,
            ..player_card
        });
    }

    // Check if all attacks are beaten
    let remaining_pending = count_pending_draws(ctx, turn_id);
    if remaining_pending == 0 {
        // All attacks beaten - defender wins the turn
        finish_turn_defender_won(ctx, game_id, turn_id)?;
    }

    log::info!("Player {:?} defended with {:?} of {:?}", 
               ctx.sender, card.rank, card.suit);
    Ok(())
}

#[reducer]
/// Defender takes all cards on the table (gives up defending)
pub fn take_cards(ctx: &ReducerContext, game_id: u64, turn_id: u64) -> Result<(), String> {
    // Validate game exists and is active
    let game = ctx.db.game().id().find(game_id)
        .ok_or("Game not found")?;
    
    if game.status != GameStatus::Active {
        return Err("Game is not active".to_string());
    }

    // Get the turn
    let turn = ctx.db.turn().id().find(turn_id)
        .ok_or("Turn not found")?;
    
    if turn.defender != ctx.sender {
        return Err("You are not the defender for this turn".to_string());
    }

    if turn.status != TurnStatus::Active {
        return Err("Turn is not active".to_string());
    }

    // Mark all draws as taken
    let draws: Vec<Draw> = ctx.db.draw()
        .iter()
        .filter(|draw| draw.turn_id == turn_id)
        .collect();

    for draw in draws {
        ctx.db.draw().id().update(Draw {
            status: DrawStatus::Taken,
            ..draw
        });
    }

    // Move all cards on table to defender's hand
    let table_cards: Vec<PlayerCard> = ctx.db.player_card()
        .iter()
        .filter(|pc| pc.game_id == game_id && pc.location == CardLocation::OnTable)
        .collect();

    for player_card in table_cards {
        ctx.db.player_card().id().update(PlayerCard {
            player: ctx.sender,
            location: CardLocation::Hand,
            ..player_card
        });
    }

    // Finish turn - defender took cards
    ctx.db.turn().id().update(Turn {
        status: TurnStatus::DefenderTook,
        finished_at: Some(ctx.timestamp),
        ..turn
    });

    // Refill hands and start next turn
    refill_hands(ctx, game_id)?;
    start_next_turn_after_take(ctx, game_id, turn.round_id)?;

    log::info!("Player {:?} took all cards", ctx.sender);
    Ok(())
}

#[reducer]
/// Pass turn (attacker cannot or chooses not to add more cards)
pub fn pass_turn(ctx: &ReducerContext, game_id: u64) -> Result<(), String> {
    // Validate game exists and is active
    let game = ctx.db.game().id().find(game_id)
        .ok_or("Game not found")?;
    
    if game.status != GameStatus::Active {
        return Err("Game is not active".to_string());
    }

    // Get current round
    let round = get_current_round(ctx, game_id)
        .ok_or("No active round found")?;

    // Get current turn
    let turn = get_active_turn(ctx, round.id)
        .ok_or("No active turn found")?;

    // Check if there are any pending attacks
    let pending_draws = count_pending_draws(ctx, turn.id);
    if pending_draws > 0 {
        return Err("Cannot pass while there are undefended attacks".to_string());
    }

    // Only attackers can pass (or anyone if anyone_can_attack is true)
    let settings = get_game_settings_for_game(ctx, game_id)?;
    if !settings.anyone_can_attack && turn.attacker != ctx.sender {
        return Err("Only the attacker can pass".to_string());
    }

    // Turn is implicitly finished when all attacks are defended and no more attacks come
    // This is handled by a timeout or explicit pass
    log::info!("Player {:?} passed turn", ctx.sender);
    Ok(())
}

// Turn Resolution Helpers

/// Finish turn when defender successfully beat all attacks
fn finish_turn_defender_won(ctx: &ReducerContext, game_id: u64, turn_id: u64) -> Result<(), String> {
    let turn = ctx.db.turn().id().find(turn_id)
        .ok_or("Turn not found")?;

    // Update turn status
    ctx.db.turn().id().update(Turn {
        status: TurnStatus::DefenderBeat,
        finished_at: Some(ctx.timestamp),
        ..turn
    });

    // Move all cards on table to discard pile
    let table_cards: Vec<PlayerCard> = ctx.db.player_card()
        .iter()
        .filter(|pc| pc.game_id == game_id && pc.location == CardLocation::OnTable)
        .collect();

    for player_card in table_cards {
        ctx.db.player_card().id().update(PlayerCard {
            location: CardLocation::Discarded,
            ..player_card
        });
    }

    // Refill hands
    refill_hands(ctx, game_id)?;

    // Check if round ended (someone emptied their hand)
    if check_round_end(ctx, game_id, turn.round_id)? {
        return Ok(());
    }

    // Start next turn with defender as new attacker
    start_next_turn_after_defense(ctx, game_id, turn.round_id, turn.defender)?;

    Ok(())
}

/// Start next turn after defender took cards (skips defender)
fn start_next_turn_after_take(ctx: &ReducerContext, game_id: u64, round_id: u64) -> Result<(), String> {
    let _game = ctx.db.game().id().find(game_id)
        .ok_or("Game not found")?;

    let last_turn = ctx.db.turn()
        .iter()
        .filter(|t| t.round_id == round_id)
        .max_by_key(|t| t.turn_number)
        .ok_or("No previous turn found")?;

    // Check if round ended
    if check_round_end(ctx, game_id, round_id)? {
        return Ok(());
    }

    // Next attacker is the player after the defender (clockwise)
    let next_attacker = get_next_player_clockwise(ctx, game_id, last_turn.defender)?;
    let next_defender = get_next_player_clockwise(ctx, game_id, next_attacker)?;

    // Don't create a new turn immediately - wait for attacker to make a move
    log::info!("Next turn: {:?} can attack {:?}", next_attacker, next_defender);
    Ok(())
}

/// Start next turn after successful defense (defender becomes attacker)
fn start_next_turn_after_defense(ctx: &ReducerContext, game_id: u64, round_id: u64, new_attacker: Identity) -> Result<(), String> {
    // Check if round ended
    if check_round_end(ctx, game_id, round_id)? {
        return Ok(());
    }

    let new_defender = get_next_player_clockwise(ctx, game_id, new_attacker)?;
    
    // Don't create a new turn immediately - wait for attacker to make a move
    log::info!("Next turn: {:?} can attack {:?}", new_attacker, new_defender);
    Ok(())
}

/// Get next active player in clockwise order
fn get_next_player_clockwise(ctx: &ReducerContext, game_id: u64, current_player: Identity) -> Result<Identity, String> {
    let current_user = ctx.db.user().identity().find(current_player)
        .ok_or("Current player not found")?;
    
    let _current_position = current_user.game_position
        .ok_or("Player has no game position")?;

    let game_players: Vec<User> = ctx.db.user()
        .iter()
        .filter(|user| user.current_game_id == Some(game_id) && user.player_status == Some(PlayerStatus::Active))
        .collect();

    if game_players.len() < 2 {
        return Err("Not enough active players".to_string());
    }

    // Sort by position and find next active player
    let mut sorted_players = game_players;
    sorted_players.sort_by_key(|p| p.game_position.unwrap_or(0));

    let current_index = sorted_players.iter()
        .position(|p| p.identity == current_player)
        .ok_or("Current player not found in game")?;

    let next_index = (current_index + 1) % sorted_players.len();
    Ok(sorted_players[next_index].identity)
}

/// Refill all players' hands from deck
fn refill_hands(ctx: &ReducerContext, game_id: u64) -> Result<(), String> {
    let settings = get_game_settings_for_game(ctx, game_id)?;
    let target_hand_size = settings.starting_cards as usize;

    // Get all active players sorted by position
    let mut players: Vec<User> = ctx.db.user()
        .iter()
        .filter(|user| user.current_game_id == Some(game_id) && user.player_status == Some(PlayerStatus::Active))
        .collect();
    
    players.sort_by_key(|p| p.game_position.unwrap_or(0));

    // Get deck cards
    let mut deck_cards: Vec<PlayerCard> = ctx.db.player_card()
        .iter()
        .filter(|pc| pc.game_id == game_id && pc.location == CardLocation::Deck)
        .collect();

    // Refill hands (attackers first, then defender)
    for player in players {
        let current_hand_size = get_player_cards(ctx, game_id, player.identity).len();
        let cards_needed = target_hand_size.saturating_sub(current_hand_size);

        for _ in 0..cards_needed {
            if let Some(deck_card) = deck_cards.pop() {
                ctx.db.player_card().id().update(PlayerCard {
                    player: player.identity,
                    location: CardLocation::Hand,
                    ..deck_card
                });
            } else {
                // No more cards in deck
                break;
            }
        }
    }

    Ok(())
}

/// Check if round has ended (only one player with cards)
fn check_round_end(ctx: &ReducerContext, game_id: u64, round_id: u64) -> Result<bool, String> {
    let players: Vec<User> = ctx.db.user()
        .iter()
        .filter(|user| user.current_game_id == Some(game_id) && user.player_status == Some(PlayerStatus::Active))
        .collect();

    let mut players_with_cards = Vec::new();

    for player in players {
        let hand_size = get_player_cards(ctx, game_id, player.identity).len();
        if hand_size > 0 {
            players_with_cards.push(player);
        } else {
            // Player finished this round
            ctx.db.user().identity().update(User {
                player_status: Some(PlayerStatus::Finished),
                ..player
            });
        }
    }

    if players_with_cards.len() <= 1 {
        // Round ended
        let round = ctx.db.round().id().find(round_id)
            .ok_or("Round not found")?;

        let loser = players_with_cards.first().map(|p| p.identity);

        ctx.db.round().id().update(Round {
            status: RoundStatus::Finished,
            loser,
            finished_at: Some(ctx.timestamp),
            ..round
        });

        // Handle scoring and check if game ended
        handle_round_scoring(ctx, game_id, loser)?;

        log::info!("Round {} ended, loser: {:?}", round.round_number, loser);
        return Ok(true);
    }

    Ok(false)
}

/// Handle scoring after round ends
fn handle_round_scoring(ctx: &ReducerContext, game_id: u64, loser: Option<Identity>) -> Result<(), String> {
    let settings = get_game_settings_for_game(ctx, game_id)?;

    if !settings.multi_round_mode {
        // Single round mode - game ends here
        finish_game(ctx, game_id, loser)?;
        return Ok(());
    }

    // Multi-round mode - add points and check if game should end
    if let Some(loser_identity) = loser {
        let loser_user = ctx.db.user().identity().find(loser_identity)
            .ok_or("Loser not found")?;

        let new_points = loser_user.total_points.unwrap_or(0) + 5; // 5 points for losing a round

        ctx.db.user().identity().update(User {
            total_points: Some(new_points),
            ..loser_user
        });

        // Check if player reached max points (becomes the "Fool")
        if new_points >= settings.max_points {
            finish_game(ctx, game_id, Some(loser_identity))?;
            return Ok(());
        }
    }

    // Start new round
    start_new_round(ctx, game_id)?;
    Ok(())
}

/// Start a new round
fn start_new_round(ctx: &ReducerContext, game_id: u64) -> Result<(), String> {
    let game = ctx.db.game().id().find(game_id)
        .ok_or("Game not found")?;

    let new_round_number = game.current_round + 1;
    let round_id = generate_round_id(game_id, new_round_number);

    // Reset all players to active
    let players: Vec<User> = ctx.db.user()
        .iter()
        .filter(|user| user.current_game_id == Some(game_id))
        .collect();

    for player in players {
        ctx.db.user().identity().update(User {
            player_status: Some(PlayerStatus::Active),
            ..player
        });
    }

    // Create new round
    ctx.db.round().insert(Round {
        id: round_id,
        game_id,
        round_number: new_round_number,
        status: RoundStatus::Active,
        loser: None,
        started_at: ctx.timestamp,
        finished_at: None,
    });

    // Update game
    ctx.db.game().id().update(Game {
        current_round: new_round_number,
        ..game
    });

    // Redeal cards (simplified - would need proper shuffle and deal logic)
    log::info!("Started new round {} for game {}", new_round_number, game_id);
    Ok(())
}

/// Finish the entire game
fn finish_game(ctx: &ReducerContext, game_id: u64, final_loser: Option<Identity>) -> Result<(), String> {
    let game = ctx.db.game().id().find(game_id)
        .ok_or("Game not found")?;

    ctx.db.game().id().update(Game {
        status: GameStatus::Finished,
        finished_at: Some(ctx.timestamp),
        ..game
    });

    // Reset all players' game state
    let players: Vec<User> = ctx.db.user()
        .iter()
        .filter(|user| user.current_game_id == Some(game_id))
        .collect();

    for player in players {
        ctx.db.user().identity().update(User {
            current_game_id: None,
            game_position: None,
            total_points: None,
            player_status: None,
            ..player
        });
    }

    // Update lobby status
    ctx.db.lobby().id().update(Lobby {
        status: LobbyStatus::Finished,
        ..ctx.db.lobby().id().find(game.lobby_id).unwrap()
    });

    log::info!("Game {} finished, final loser: {:?}", game_id, final_loser);
    Ok(())
}

// Additional Query Functions

/// Get current turn for a game
pub fn get_current_turn(ctx: &ReducerContext, game_id: u64) -> Option<Turn> {
    if let Some(round) = get_current_round(ctx, game_id) {
        get_active_turn(ctx, round.id)
    } else {
        None
    }
}

/// Get all draws for a turn
pub fn get_turn_draws(ctx: &ReducerContext, turn_id: u64) -> Vec<Draw> {
    ctx.db.draw()
        .iter()
        .filter(|draw| draw.turn_id == turn_id)
        .collect()
}

/// Get cards currently on the table
pub fn get_table_cards(ctx: &ReducerContext, game_id: u64) -> Vec<PlayerCard> {
    ctx.db.player_card()
        .iter()
        .filter(|pc| pc.game_id == game_id && pc.location == CardLocation::OnTable)
        .collect()
}
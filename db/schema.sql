-- An application user who can access the app. Does not include profile information.k
create table if not exists users
(
    id bigserial primary key,
    -- login name, usually an email address for an internal user.
    name text not null,
    -- the application is responsible for hashing the password
    password text,
    status text check ( status in ('verifying-email', 'normal', 'reset-password', 'disabled') ) not null,
    -- a random string generated by application and used for verifying email address or resetting password.
    challenge text
);

create unique index if not exists users_name_index on users (name);

-- An AI personality with its own system context prompt and voice model.
create table if not exists ai_persons
(
    id bigserial primary key,
    user_id bigint references users (id) on delete cascade not null,
    name text not null,
    -- Contextual, background information for the system role, e.g. you are Esther in Shushan.
    context_prompt text not null
);
create index if not exists ai_persons_user_id_index on ai_persons (user_id);

-- Voice recording samples of an AI personality.
create table if not exists voice_samples
(
    id bigserial primary key,
    ai_person_id bigint references ai_persons (id) on delete cascade not null,
    file_name text,
    timestamp timestamp with time zone not null
);
create index if not exists voice_sample_ai_person_id_index on voice_samples (ai_person_id);

-- Voice models of an AI personality.
create table if not exists voice_models
(
    id bigserial primary key,
    voice_sample_id bigint references voice_samples (id) on delete cascade not null,
    -- Whether a model has been created for the sample yet.
    status text check ( status in ('processing', 'ready') ) not null,
    file_name text,
    timestamp timestamp with time zone not null
);
create index if not exists voice_model_sample_id_index on voice_models (voice_sample_id);

--- The user's side of conversation with an AI personality - a voice note or text message intended for an AI personality.
create table if not exists user_prompts
(
    id bigserial primary key,
    ai_person_id bigint references ai_persons (id) on delete cascade not null,
    timestamp timestamp with time zone not null
    -- The user prompt content is either a text message or a voice message.
);

create index if not exists user_prompt_ai_person_id_index on user_prompts (ai_person_id);

-- The user's side of conversation - a plain text message.
create table if not exists user_text_prompts
(
    id bigserial primary key,
    user_prompt_id bigint references user_prompts (id) on delete cascade not null,
    message text not null
);
create index if not exists user_text_prompt_id_index on user_text_prompts (user_prompt_id);

-- The user's side of conversation - a voice message.
create table if not exists user_voice_prompts
(
    id bigserial primary key,
    user_prompt_id bigint references user_prompts (id) on delete cascade not null,
    -- Whether this voice note has been transcribed into text.
    status text check ( status in ('processing', 'ready') ) not null,
    file_name text not null,
    transcription text
);
create index if not exists user_voice_prompt_id_index on user_voice_prompts (user_prompt_id);

--- The AI personality's side of conversation - AI's replies to user's prompts.
create table if not exists ai_person_replies
(
    id bigserial primary key,
    user_prompt_id bigint references ai_persons (id) on delete cascade not null,
    -- Whether LLM has generated a reply in response to the prompt.
    status text check ( status in ('processing', 'ready') ) not null,
    message text not null,
    timestamp timestamp with time zone not null
);
create index if not exists ai_person_reply_person_id_index on ai_person_replies (user_prompt_id);

-- The AI personality's side of conversation - AI's reply in cloned voice model.
create table if not exists ai_person_reply_voices
(
    id bigserial primary key,
    ai_person_reply_id bigint references ai_person_replies (id) on delete cascade not null,
    status text check ( status in ('processing', 'ready') ) not null,
    file_name text
);
create index if not exists ai_person_reply_voice_reply_id_index  on ai_person_reply_voices (ai_person_reply_id);

-- name: CreateUser :one
insert into users (name, password, status) values ($1, $2, $3) returning *;
-- name: ListUsers :many
select * from users;
-- name: GetUserByName :one
select * from users where name = $1 limit 1;

-- name: CreateAIPerson :one
insert into ai_persons (user_id, name, context_prompt) values ($1, $2, $3) returning *;
-- name: ListAIPersons :many
select * from ai_persons where user_id = $1 order by id;
-- name: GetAIPerson :one
select * from ai_persons where id = $1;
-- name: UpdateAIPersonContextPromptByID :exec
update ai_persons set context_prompt = $1 where id = $2;

-- name: CreateVoiceSample :one
insert into voice_samples (ai_person_id, file_name, timestamp) values ($1, $2, $3) returning *;
-- name: GetVoiceSampleByID :one
select * from voice_samples where id = $1 limit 1;
-- name: ListVoiceSamples :many
select * from voice_samples where ai_person_id = $1 order by id;

-- name: CreateVoiceModel :one
insert into voice_models (voice_sample_id, status, file_name, timestamp) values ($1, $2, $3, $4) returning *;
-- name: GetVoiceModelByVoiceSample :one
select * from voice_models where voice_sample_id = $1;
-- name: GetLatestVoiceModel :one
select m.id as id, m.status as status, m.file_name as file_name, m.timestamp as timestamp,
a.user_id as user_id, a.name as ai_name, a.context_prompt as ai_context_prompt
from voice_models m
join voice_samples s on m.voice_sample_id = s.id
join ai_persons a on s.ai_person_id = a.id and a.id = $1
order by m.timestamp desc
limit 1;
-- name: UpdateVoiceModelByID :exec
update voice_models set status = $1 where id = $2;

-- name: CreateUserPrompt :one
insert into user_prompts (ai_person_id, timestamp) values ($1, $2) returning *;

-- name: CreateUserTextPrompt :one
insert into user_text_prompts (user_prompt_id, message) values ($1, $2) returning *;

-- name: CreateUserVoicePrompt :one
insert into user_voice_prompts (user_prompt_id, status, file_name, transcription) values ($1, $2, $3, $4) returning *;
-- name: UpdateUserVoicePromptStatusByID :exec
update user_voice_prompts set status = $1 where id = $2;

-- name: CreateAIPersonReply :one
insert into ai_person_replies (user_prompt_id, status, message, timestamp) values ($1, $2, $3, $4) returning *;
-- name: UpdateAIPersonReplyByID :exec
update ai_person_replies set status = $1 and message = $2 where id = $3;

-- name: CreateAIPersonReplyVoice :one
insert into ai_person_reply_voices (ai_person_reply_id, status, file_name) values ($1, $2, $3) returning *;
-- name: UpdateAIPersonReplyVoiceStatusByID :exec
update ai_person_reply_voices set status = $1 where id = $2;

-- name: ListConversations :many
select u.id as id, u.ai_person_id as ai_person_id, u.timestamp as timestamp,
t.message as text_message,
v.status as voice_status, v.file_name as voice_filename, v.transcription as voice_transcription,
r.status as reply_status, r.message as reply_message, r.timestamp as reply_timestamp,
rv.status as reply_voice_status, rv.file_name as reply_voice_filename
from user_prompts u
left outer join user_text_prompts t on t.user_prompt_id = u.id
left outer join user_voice_prompts v on v.user_prompt_id = u.id
left outer join ai_person_replies r on r.user_prompt_id = u.id
left outer join ai_person_reply_voices rv on rv.ai_person_reply_id = r.id
where ai_person_id = $1;

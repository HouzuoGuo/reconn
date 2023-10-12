-- name: CreateUser :one
insert into users (name, password, status) values ($1, $2, $3) returning *;
-- name: GetUserByName :one
select * from users where name = $1 limit 1;


-- name: CreateAIPerson :one
insert into ai_persons (user_id, name, context_prompt) values ($1, $2, $3) returning *;
-- name: ListAIPersons :many
select * from ai_persons where user_id = $1 order by id;
-- name: UpdateAIPersonContextPromptByID :exec
update ai_persons set context_prompt = $1 where id = $2;

-- name: CreateVoiceSample :one
insert into voice_samples (ai_person_id, file_name, timestamp) values ($1, $2, $3) returning *;
-- name: ListVoiceSamples :many
select * from voice_samples where ai_person_id = $1 order by id;

-- name: CreateVoiceModel :one
insert into voice_models (voice_sample_id, status, file_name, timestamp) values ($1, $2, $3, $4) returning *;
-- name: GetVoiceModelByVoiceSample :one
select * from voice_models where voice_sample_id = $1;
-- name: UpdateVoiceModelStatusByID :exec
update voice_models set status = $1 where id = $2;

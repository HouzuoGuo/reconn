import { HttpClient, HttpResponse } from '@angular/common/http';
import { EventEmitter, Injectable, NgModule } from '@angular/core';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';

export interface ReadbackResponse {
  address: string;
  headers: any;
  method: string;
  url: string;
}

@Injectable()
export class ReadbackService {
  constructor(private http: HttpClient) {}
  readback(): Observable<ReadbackResponse> {
    return this.http.get<ReadbackResponse>("/api/debug/readback");
  }
}

export interface SinglePromptResponse {
  reply: string;
}

export interface TranscribeRealTimeResponse {
  language: string;
  content: string;
}

export interface VoiceModelResponse {
  models: Map<string, VoiceModel>;
}

export interface VoiceModel {
  fileName: string;
  userId: string;
  lastModified: string;
}

export interface CloneRealtimeResponse {
  model: string;
}

export interface SqlNullString {
  String?: string;
  Valid?: boolean;
}

export interface User {
  ID?: number;
  Name?: string;
  Password?: SqlNullString;
  Status?: string;
  Challenge?: SqlNullString;
}

export interface AIPerson {
  ID?: number;
  UserID?: number;
  Name?: string;
  ContextPrompt?: string;
}

export interface UpdateAIPersonContextPromptByIDParams {
  ID?: number;
  ContextPrompt?: string;
}

@Injectable()
export class ChatService {
  constructor(readonly http: HttpClient) {}

  // Debug AI & LLM interaction endpoints.
  cloneRealTime(userID: string, blob: Blob): Observable<CloneRealtimeResponse> {
    return this.http.post<CloneRealtimeResponse>("/api/debug/clone-rt/" + userID, blob, { headers: { 'content-type': 'audio/wav' } });
  }

  textToSpeechRealTime(userID: string, text: string, topK: number, topP: number, mineosP: number, semanticTemp: number, waveformTemp: number, fineTemp: number): Observable<Blob> {
    return this.http.post("/api/debug/tts-rt/" + userID, { text, topK, topP, mineosP, semanticTemp, waveformTemp, fineTemp }, { headers: { 'content-type': 'application/json' }, responseType: 'blob' });
  }

  listVoiceModel(): Observable<VoiceModelResponse> {
    return this.http.get<VoiceModelResponse>("/api/debug/voice-model");
  }

  converseSinglePrompt(userID: string, systemPrompt: string, userPrompt: string): Observable<SinglePromptResponse> {
    return this.http.post<SinglePromptResponse>("/api/debug/converse-single-prompt/" + userID, JSON.stringify({
      'systemPrompt': systemPrompt,
      'userPrompt': userPrompt,
    }), { headers: { 'content-type': 'application/json' } });
  }

  transcribeRealTime(userID: string, blob: Blob): Observable<TranscribeRealTimeResponse> {
    return this.http.post<TranscribeRealTimeResponse>("/api/debug/transcribe-rt/" + userID, blob, { headers: { 'content-type': 'audio/wav' } });
  }

  // Debug user endpoints.
  createUser(user: User): Observable<User> {
    return this.http.post<User>("/api/debug/user", user, { headers: { 'content-type': 'application/json' } });
  }
  listUsers(): Observable<User[]> {
    return this.http.get<User[]>("/api/debug/user",);
  }
  getUserByName(name: string): Observable<User> {
    return this.http.get<User>("/api/debug/user/" + name);
  }
  // Debug AI person endpoints.
  createAIPerson(aiPerson: AIPerson): Observable<AIPerson> {
    return this.http.post<AIPerson>("/api/debug/ai_person", aiPerson, { headers: { 'content-type': 'application/json' } });
  }
  listAIPersons(userID: number): Observable<AIPerson[]> {
    return this.http.get<AIPerson[]>("/api/debug/user/" + userID + "/ai_person");
  }
  updateAIPerson(aiPersonID: number, params: UpdateAIPersonContextPromptByIDParams) {
    return this.http.put<AIPerson[]>("/api/debug/ai_person/" + aiPersonID, params, { headers: { 'content-type': 'application/json' } });
  }
  // Debug voice sample and model endpoints.
  // Debug conversations.

}
@NgModule({
  declarations: [],
  imports: [],
  exports: [],
  providers: [
    ChatService,
    ReadbackService,
  ]
})
export class ChatModule {}

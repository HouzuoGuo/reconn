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
    return this.http.get<ReadbackResponse>("/api/debug/greadback");
  }
}

export interface SinglePromptResponse {
  reply: string;
}

export interface TranscribeRealTimeResponse {
  language: string;
  content: string;
}

@Injectable()
export class ChatService {
  constructor(readonly http: HttpClient) {}

  converseSinglePrompt(userID: string, systemPrompt: string, userPrompt: string): Observable<SinglePromptResponse> {
    return this.http.post<SinglePromptResponse>("/api/debug/converse-single-prompt/" + userID, JSON.stringify({
      'systemPrompt': systemPrompt,
      'userPrompt': userPrompt,
    }), { headers: { 'content-type': 'application/json' } });
  }

  transcribeRealTime(userID: string, blob: Blob): Observable<TranscribeRealTimeResponse> {
    return this.http.post<TranscribeRealTimeResponse>("/api/debug/transcribe-rt/" + userID, blob, { headers: { 'content-type': 'audio/wav' } });
  }
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

@Injectable()
export class VoiceService {
  constructor(private http: HttpClient) {}
  listVoiceModel(): Observable<VoiceModelResponse> {
    return this.http.get<VoiceModelResponse>("/api/debug/voice-model");
  }

  cloneRealTime(userID: string, blob: Blob): Observable<CloneRealtimeResponse> {
    return this.http.post<CloneRealtimeResponse>("/api/debug/clone-rt/" + userID, blob, { headers: { 'content-type': 'audio/wav' } });
  }

  textToSpeechRealTime(userID: string, text: string, topK: number, topP: number, mineosP: number, semanticTemp: number, waveformTemp: number, fineTemp: number): Observable<Blob> {
    return this.http.post("/api/debug/tts-rt/" + userID, { text, topK, topP, mineosP, semanticTemp, waveformTemp, fineTemp }, { headers: { 'content-type': 'application/json' }, responseType: 'blob' });
  }
}

@NgModule({
  declarations: [],
  imports: [],
  exports: [],
  providers: [
    ChatService,
    VoiceService,
    ReadbackService,
  ]
})
export class ChatModule {}

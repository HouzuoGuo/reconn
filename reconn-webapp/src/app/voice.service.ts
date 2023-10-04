import { Injectable } from '@angular/core';
import { HttpClient, HttpResponse } from '@angular/common/http';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';

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
    return this.http.get<VoiceModelResponse>("/api/voice-model");
  }

  realTimeClone(userID: string, blob: Blob): Observable<CloneRealtimeResponse> {
    return this.http.post<CloneRealtimeResponse>("/api/clone-rt/" + userID, blob, { headers: { 'content-type': 'audio/wav' } });
  }

  realTimeTTS(userID: string, text: string, topK: number, topP: number, mineosP: number, semanticTemp: number, waveformTemp: number, fineTemp: number): Observable<Blob> {
    return this.http.post("/api/tts-rt/" + userID, { text, topK, topP, mineosP, semanticTemp, waveformTemp, fineTemp }, { headers: { 'content-type': 'application/json' }, responseType: 'blob' });
  }
}

import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

export interface VoiceModelResponse {
  models: Map<string, VoiceModel>;
}

export interface VoiceModel {
  fileName: string;
  userId: string;
}


@Injectable()
export class VoiceService {
  constructor(private http: HttpClient) {}
  listVoiceModel(): Observable<VoiceModelResponse> {
    return this.http.get<VoiceModelResponse>("/api/voice-model");
  }
}

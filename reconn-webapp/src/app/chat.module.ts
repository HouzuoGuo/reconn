import { HttpClient, HttpResponse } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';

export interface SinglePromptResponse {
  reply: string
}

@Injectable()
export class ChatService {
  constructor(readonly http: HttpClient) {}

  converseSinglePrompt(userID: string, systemPrompt: string, userPrompt: string): Observable<SinglePromptResponse> {
    return this.http.post<SinglePromptResponse>("/api/converse-single-prompt/:user_id" + userID, JSON.stringify({
      'systemPrompt': systemPrompt,
      'userPrompt': userPrompt,
    }), { headers: { 'content-type': 'application/json' } });
  }
}

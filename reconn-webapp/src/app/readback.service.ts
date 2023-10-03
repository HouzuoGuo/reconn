import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

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
    return this.http.get<ReadbackResponse>("/api/readback");
  }
}

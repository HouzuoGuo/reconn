import { HttpErrorResponse } from '@angular/common/http';
import { Component, ElementRef, OnInit, ViewChild } from '@angular/core';
import { EMPTY, Observable, ReplaySubject, fromEvent, of, shareReplay } from 'rxjs';
import { catchError, exhaustMap, map, startWith, tap } from 'rxjs/operators';
import { AIPerson, ChatService, ReadbackResponse, ReadbackService, UpdateAIPersonContextPromptByIDParams, VoiceModelResponse } from './chat.module';

@Component({
  selector: 'conversation-mgmt',
  templateUrl: './conversation-mgmt.component.html',
})
export class ConversationManagementComponent implements OnInit {
  constructor(private chatService: ChatService) {
  }

  ngOnInit() {
  }
}

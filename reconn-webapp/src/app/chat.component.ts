import { HttpErrorResponse } from '@angular/common/http';
import { Component, ElementRef, OnInit, ViewChild } from '@angular/core';
import { Observable, ReplaySubject, combineLatest, fromEvent, of, shareReplay } from 'rxjs';
import { catchError, exhaustMap, map, startWith, switchMap, tap } from 'rxjs/operators';
import { AudioRecorderService, ErrorCase, OutputFormat } from './audio_recorder.module';
import { ChatService, SinglePromptResponse } from './chat.module';
import { ReadbackResponse, ReadbackService } from './readback.service';
import { CloneRealtimeResponse, VoiceModelResponse, VoiceService } from './voice.service';

@Component({
  selector: 'chat',
  templateUrl: './chat.component.html',
})
export class ChatComponent implements OnInit {
  @ViewChild('singlePromptButton', { static: true }) singlePromptButton!: ElementRef;

  userID = '';
  systemPrompt = 'You are a helpful AI assistant.';
  userPrompt = 'When is the next Purim?'
  singlePromptResponse!: Observable<SinglePromptResponse>;

  constructor(readonly chatService: ChatService) {
  }

  ngOnInit() {
    this.singlePromptResponse = fromEvent(this.singlePromptButton.nativeElement, 'click')
      .pipe(
        exhaustMap((click) => this.chatService.converseSinglePrompt(this.userID, this.systemPrompt, this.userPrompt)),
        shareReplay({ bufferSize: 1, refCount: true })
      );
  }

  singlePromptClick() {
  }
}

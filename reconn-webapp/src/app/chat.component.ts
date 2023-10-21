import { HttpErrorResponse } from '@angular/common/http';
import { Component, ElementRef, OnInit, ViewChild } from '@angular/core';
import { Observable, ReplaySubject, combineLatest, fromEvent, of, shareReplay } from 'rxjs';
import { catchError, exhaustMap, map, startWith, switchMap, tap } from 'rxjs/operators';
import { AudioRecorderService, ErrorCase, OutputFormat } from './audio_recorder.module';
import { ChatService, CloneRealtimeResponse, ReadbackResponse, ReadbackService, SinglePromptResponse, VoiceModelResponse } from './chat.module';

@Component({
  selector: 'chat',
  templateUrl: './chat.component.html',
})
export class ChatComponent implements OnInit {
  @ViewChild('singlePromptButton', { static: true }) singlePromptButton!: ElementRef;

  systemPrompt = 'You are a helpful AI assistant.';
  userPrompt = 'What food is popular on Purim? What is the date in 2024?'
  singlePromptResponse!: Observable<SinglePromptResponse>;

  constructor(readonly chatService: ChatService) {
  }

  ngOnInit() {
    this.singlePromptResponse = fromEvent(this.singlePromptButton.nativeElement, 'click')
      .pipe(
        exhaustMap((click) => this.chatService.converseSinglePrompt(this.systemPrompt, this.userPrompt)),
        shareReplay({ bufferSize: 1, refCount: true })
      );
  }

  singlePromptClick() {
  }
}

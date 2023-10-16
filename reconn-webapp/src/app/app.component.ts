import { Component, ElementRef, OnInit, ViewChild } from '@angular/core';
import { Observable, fromEvent, of, shareReplay } from 'rxjs';
import { exhaustMap, startWith } from 'rxjs/operators';
import { ReadbackResponse, ReadbackService, VoiceModelResponse, ChatService } from './chat.module';

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css']
})
export class AppComponent implements OnInit {
  readonly Object = Object;
  @ViewChild('refreshVoiceModelsButton', { static: true }) refreshVoiceModelsButton!: ElementRef;

  readbackResp: Observable<ReadbackResponse>;
  voiceModelListResp: Observable<VoiceModelResponse | undefined> = of(undefined);

  constructor(readonly readbackService: ReadbackService, readonly chatService: ChatService) {
    this.readbackResp = readbackService.readback().pipe(shareReplay({ bufferSize: 1, refCount: true }));
  }

  ngOnInit() {
    this.voiceModelListResp = fromEvent(this.refreshVoiceModelsButton.nativeElement, 'click')
      .pipe(
        startWith(undefined),
        exhaustMap((click) => this.chatService.listVoiceModel()),
        shareReplay({ bufferSize: 1, refCount: true })
      );
  }
}

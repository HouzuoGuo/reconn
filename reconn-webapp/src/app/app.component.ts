import { Component, ElementRef, OnInit, ViewChild } from '@angular/core';
import { Observable, fromEvent, of, shareReplay } from 'rxjs';
import { exhaustMap, startWith } from 'rxjs/operators';
import { ReadbackResponse, ReadbackService } from './readback.service';
import { VoiceModelResponse, VoiceService } from './voice.service';

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

  constructor(private readbackService: ReadbackService, private voiceService: VoiceService) {
    this.readbackResp = readbackService.readback().pipe(shareReplay({ bufferSize: 1, refCount: true }));
  }

  ngOnInit() {
    this.voiceModelListResp = fromEvent(this.refreshVoiceModelsButton.nativeElement, 'click')
      .pipe(
        startWith(undefined),
        exhaustMap((click) => this.voiceService.listVoiceModel()),
        shareReplay({ bufferSize: 1, refCount: true })
      );
  }
}
